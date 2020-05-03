package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "net/http"
)

type ProjectStats struct {
    ProjectId      string  `json:"id"`
    InternalId     int     `json:"internal_id"`
    Type           string  `json:"type"`
    NetworkComp    float32 `json:"network_comp"`
    CodeChurn      float32 `json:"code_churn"`
    NetworkCompMsg string  `json:"network_comp_msg"`
    CodeChurnMsg   string  `json:"code_churn_msg"`
}

type PackageStats struct {
    PackageId   string  `json:"id"`
    PackageName string  `json:"name"`
    Type        string  `json:"type"`
    InternalId  int     `json:"internal_id"`
    NetworkComp float32 `json:"network_comp"`
}

type ClassOrInterfaceStats struct {
    ClassOrInterfaceId   string  `json:"id"`
    ClassOrInterfaceName string  `json:"name"`
    Type                 string  `json:"type"`
    InternalId           int     `json:"internal_id"`
    NetworkComp          float32 `json:"network_comp"`
}

type MethodStats struct {
    MethodId    string  `json:"id"`
    MethodName  string  `json:"name"`
    Type        string  `json:"type"`
    InternalId  int     `json:"internal_id"`
    NetworkComp float32 `json:"network_comp"`
}

type Project struct {
    Project           ProjectStats            `json:"Project"`
    Packages          []PackageStats          `json:"Package"`
    ClassOrInterfaces []ClassOrInterfaceStats `json:"ClassOrInterface"`
    Methods           []MethodStats           `json:"Method"`
}

func getMetrics(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    owner, ownerSet := vars["owner"]
    if !ownerSet {
        print("Error getting owner.")
        return
    }
    repo, repoSet := vars["repo"]
    if !repoSet {
        print("Error getting repo.")
        return
    }
    projectName := owner + "/" + repo
    fmt.Println("Get metrics request for: " + projectName)

    // Check if project already in cache
    if project, found := getProjectFromCache(projectName); found {
        writeErr := json.NewEncoder(w).Encode(project)
        if writeErr != nil {
            fmt.Println("Write error:", writeErr)
        }
        return
    }

    // Check if parsing in progress
    if status, found := getProjectStatus(projectName); found && status.Status != STATUS_ERROR {
        writeErr := json.NewEncoder(w).Encode(map[string]string{
            "status": status.Status,
            "msg":    status.Msg,
        })
        if writeErr != nil {
            fmt.Println("Write error:", writeErr)
        }
        return
    }

    // Not in cache or in progress, try and get from SNA (already parsed and in DB)
    projectFetched := getProject(owner, repo)
    if projectFetched {
        project, _ := getProjectFromCache(projectName)
        writeErr := json.NewEncoder(w).Encode(project)
        if writeErr != nil {
            fmt.Println("Write error:", writeErr)
        }
        return
    }

    // Project not yet parsed. Attempt to trigger analysis.
    go triggerAnalysis(owner, repo)
    w.WriteHeader(http.StatusAccepted)
    writeErr := json.NewEncoder(w).Encode(map[string]string{
        "status": "initiating_parsing",
        "msg":    "Analysing project.",
    })
    if writeErr != nil {
        fmt.Println("Write error:", writeErr)
    }
}

func getProject(owner, repo string) bool {
    projectName := owner + "/" + repo
    resp, err := http.Get(snaUrl + "/projects/" + owner + "/" + repo)
    if err != nil {
        fmt.Println("Error:", err)
    }
    if resp.StatusCode == http.StatusOK {
        var project Project
        err := json.NewDecoder(resp.Body).Decode(&project)
        if err != nil {
            fmt.Println("Error:", err)
        }
        setProjectCache(projectName, project)
        setProjectStatus(projectName, Status{Status: STATUS_COMPLETE})
        return true
    }
    return false
}

func triggerAnalysis(owner, repo string) {
    projectName := owner + "/" + repo
    fmt.Println("Requesting project parsing for:", projectName)
    // Blocks until response received
    bodyJson := map[string]string{"owner": owner, "repo": repo}
    fmt.Println("body owner:", owner, "repo:", repo)

    bodyBytes, marshallErr := json.Marshal(bodyJson)
    if marshallErr != nil {
        fmt.Println("Error marshalling body: ", marshallErr)
    }
    resp, err := http.Post(snaUrl+"/analyse", "application/json", bytes.NewBuffer(bodyBytes))
    if err != nil {
        fmt.Println("Error:", err)
    }
    if resp.StatusCode == http.StatusInternalServerError {
        setProjectStatus(owner+"/"+repo, Status{Status: STATUS_ERROR})
        return
    } else if resp.StatusCode == http.StatusNotFound {
        setProjectStatus(owner+"/"+repo, Status{Status: STATUS_NOT_FOUND})
        return
    } else if resp.StatusCode == http.StatusServiceUnavailable {
        setProjectStatus(owner+"/"+repo, Status{Status: STATUS_CANNOT_PARSE, Msg: "Project cannot be analysed as it has not been parsed and system is running in standalone mode."})
        return
    }
    var project Project
    parseErr := json.NewDecoder(resp.Body).Decode(&project)
    if parseErr != nil {
        fmt.Println("Error:", parseErr)
    }
    setProjectCache(projectName, project)
    setProjectStatus(projectName, Status{Status: STATUS_COMPLETE})
    fmt.Println(resp)
}
