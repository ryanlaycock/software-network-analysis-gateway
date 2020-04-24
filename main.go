package main

import (
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "net/http"
    "os"
    "sync"
)

const (
    STATUS_ERROR     = "error"
    STATUS_NOT_FOUND = "not_found"
    STATUS_COMPLETE  = "complete"
)

var (
    projectStatuses ProjectStatuses
    projectCache    ProjectCache
    snaUrl          string
)

type Status struct {
    Status string `json:"status"`
    Msg    string `json:"msg"`
}

type ProjectCache struct {
    Mutex    sync.Mutex
    Projects map[string]Project
}

type ProjectStatuses struct {
    Mutex  sync.Mutex
    Status map[string]Status
}

type IncomingStatus struct {
    Status      string `json:"status"`
    Msg         string `json:"msg"`
    ProjectName string `json:"project_name"`
}

type ArtifactStats struct {
    ArtifactId    string  `json:"id"`
    ArtifactName  string  `json:"name"`
    ArtifactGroup string  `json:"group"`
    Type          string  `json:"type"`
    InternalId    string  `json:"internal_id"`
    PageRank      float32 `json:"page_rank"`
}

type Artifacts struct {
    Artifacts              []ArtifactStats `json:"artifacts"`
    DirectDependencies     []ArtifactStats `json:"direct_dependencies"`
    TransitiveDependencies []ArtifactStats `json:"transitive_dependecies"`
    Dependents             []ArtifactStats `json:"dependents"`
}

func updateProjectStatus(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Status update")
    var incomingStatus IncomingStatus
    err := json.NewDecoder(r.Body).Decode(&incomingStatus)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    fmt.Println("Update: " + incomingStatus.Status)
    setProjectStatus(incomingStatus.ProjectName, Status{
        Status: incomingStatus.Status,
        Msg:    incomingStatus.Msg,
    })
}

func setProjectStatus(projectName string, status Status) {
    projectStatuses.Mutex.Lock()
    projectStatuses.Status[projectName] = status
    projectStatuses.Mutex.Unlock()
}

func getProjectStatus(projectName string) (Status, bool) {
    projectStatuses.Mutex.Lock()
    status, found := projectStatuses.Status[projectName]
    projectStatuses.Mutex.Unlock()
    return status, found
}

func setProjectCache(projectName string, project Project) {
    projectCache.Mutex.Lock()
    projectCache.Projects[projectName] = project
    projectCache.Mutex.Unlock()
}

func getProjectFromCache(projectName string) (Project, bool) {
    projectCache.Mutex.Lock()
    project, found := projectCache.Projects[projectName]
    projectCache.Mutex.Unlock()
    return project, found
}

func main() {
    snaUrl = os.Getenv("SNA_ADDR")
    projectStatuses = ProjectStatuses{}
    projectStatuses.Status = map[string]Status{}
    projectStatuses.Mutex = sync.Mutex{}
    projectCache = ProjectCache{}
    projectCache.Projects = map[string]Project{}
    projectCache.Mutex = sync.Mutex{}

    router := mux.NewRouter()

    router.HandleFunc("/projects/{owner}/{repo}/metrics", getMetrics).Methods("GET")
    router.HandleFunc("/projects/{owner}/{repo}/status", updateProjectStatus).Methods("POST")

    err := http.ListenAndServe(":8080", router)
    if err != nil {
        fmt.Println(err)
    }
}
