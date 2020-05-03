package main

import (
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "net/http"
)

type ArtifactStats struct {
    ArtifactId    string  `json:"id"`
    ArtifactName  string  `json:"artifact"`
    ArtifactGroup string  `json:"group"`
    Type          string  `json:"type"`
    InternalId    int     `json:"internal_id"`
    PageRank      float32 `json:"page_rank"`
    OverallRank   int     `json:"overall_rank"`
}

type ProjectArtifactsStats struct {
    MaxRank                     int `json:"max_rank"`
    NumOfDirectDependencies     int `json:"num_of_direct_dependencies"`
    NumOfTransitiveDependencies int `json:"num_of_transitive_dependencies"`
    NumOfDependents             int `json:"num_of_dependents"`
}

type Artifacts struct {
    Artifacts              map[string]ArtifactStats `json:"Artifact"`
    DirectDependencies     map[string]ArtifactStats `json:"DirectDependency"`
    TransitiveDependencies map[string]ArtifactStats `json:"TransitiveDependency"`
    Dependents             map[string]ArtifactStats `json:"Dependent"`
}

type ArtifactsResponse struct {
    ProjectStats           ProjectArtifactsStats `json:"ProjectStats"`
    Artifacts              []ArtifactStats       `json:"Artifact"`
    DirectDependencies     []ArtifactStats       `json:"DirectDependency"`
    TransitiveDependencies []ArtifactStats       `json:"TransitiveDependency"`
    Dependents             []ArtifactStats       `json:"Dependent"`
}

type Ranks struct {
    OverallRank int     `json:"overall_rank"`
    PageRank    float32 `json:"pagerank"`
}

func getArtifacts(w http.ResponseWriter, r *http.Request) {
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
    fmt.Println("Get artifacts request for: " + projectName)

    // Check if parsing in progress
    // TODO Simplify
    if status, found := getArtifactsStatus(projectName); found && (status.Status == STATUS_IN_PROGRESS || status.Status == STATUS_NOT_FOUND) {
        if status.Status == STATUS_NOT_FOUND {
            w.WriteHeader(http.StatusNotFound)
            defer setArtifactsStatus(projectName, Status{}) // Reset so the artifacts can be searched again in the future
        } else {
            w.WriteHeader(http.StatusAccepted)
        }
        writeErr := json.NewEncoder(w).Encode(map[string]string{
            "status": status.Status,
            "msg":    status.Msg,
        })
        if writeErr != nil {
            fmt.Println("Write error:", writeErr)
        }
        return
    }

    // Check if artifacts already in cache
    if artifacts, found := getArtifactFromCache(projectName); found {
        artifactsRank := artifactsWithRank(artifacts)
        artifactsRank.ProjectStats = getProjectOverallStats(artifactsRank)
        writeErr := json.NewEncoder(w).Encode(artifactsRank)
        if writeErr != nil {
            fmt.Println("Write error:", writeErr)
        }
        return
    }

    go fetchArtifacts(owner, repo)
    status := Status{Status: STATUS_IN_PROGRESS, Msg: "Fetching artifacts."}
    setArtifactsStatus(projectName, status)
    w.WriteHeader(http.StatusAccepted)
    writeErr := json.NewEncoder(w).Encode(map[string]string{
        "status": status.Status,
        "msg":    status.Msg,
    })
    if writeErr != nil {
        fmt.Println("Write error:", writeErr)
    }

}

func artifactsWithRank(artifacts Artifacts) ArtifactsResponse {
    pageRankCache.Mutex.Lock()
    defer pageRankCache.Mutex.Unlock()
    artifactsRank := ArtifactsResponse{
        Artifacts:              addRank(artifacts.Artifacts),
        DirectDependencies:     addRank(artifacts.DirectDependencies),
        TransitiveDependencies: addRank(artifacts.TransitiveDependencies),
        Dependents:             addRank(artifacts.Dependents),
    }
    return artifactsRank
}

func getProjectOverallStats(artifacts ArtifactsResponse) ProjectArtifactsStats {
    max := -1
    for _, artifact := range artifacts.Artifacts {
        if artifact.OverallRank < max || max == -1 {
            max = artifact.OverallRank
        }
    }
    return ProjectArtifactsStats{
        MaxRank:                     max,
        NumOfDirectDependencies:     len(artifacts.DirectDependencies),
        NumOfTransitiveDependencies: len(artifacts.TransitiveDependencies),
        NumOfDependents:             len(artifacts.Dependents),
    }
}

func addRank(artifacts map[string]ArtifactStats) []ArtifactStats {
    artifactRank := []ArtifactStats{}
    for id, artifact := range artifacts {
        if pageRank, found := pageRankCache.Ranks[id]; found {
            artifactRank = append(artifactRank, ArtifactStats{
                ArtifactId:    artifact.ArtifactId,
                ArtifactName:  artifact.ArtifactName,
                ArtifactGroup: artifact.ArtifactGroup,
                Type:          artifact.Type,
                InternalId:    artifact.InternalId,
                PageRank:      pageRank.PageRank,
                OverallRank:   pageRank.OverallRank,
            })
        } else {
            print("PageRank not found for artifact id: " + id)
        }
    }
    return artifactRank
}

func fetchPageRanks() {
    fmt.Println("Fetching pageRanks")
    resp, err := http.Get(dnaUrl + "/artifacts/pageranks")
    if err != nil {
        fmt.Println("Error:", err)
    }
    if resp.StatusCode == http.StatusOK {
        var ranks map[string]Ranks
        err := json.NewDecoder(resp.Body).Decode(&ranks)
        if err != nil {
            fmt.Println("Error:", err)
        }
        fmt.Println(ranks)
        pageRankCache.Mutex.Lock()
        defer pageRankCache.Mutex.Unlock()
        pageRankCache.Ranks = ranks
        return
    }
    print("An error occurred fetching page ranks. Status code: " + string(resp.StatusCode))
}

func artifactsInCache(artifacts map[string]ArtifactStats) bool {
    pageRankCache.Mutex.Lock()
    defer pageRankCache.Mutex.Unlock()
    for artifactId, _ := range artifacts {
        if _, found := pageRankCache.Ranks[artifactId]; !found {
            return false
        }
    }
    return true
}

func fetchArtifacts(owner, repo string) bool {
    projectName := owner + "/" + repo
    resp, err := http.Get(dnaUrl + "/artifacts/" + owner + "/" + repo)
    if err != nil {
        fmt.Println("Error:", err)
    }
    if resp.StatusCode == http.StatusOK {
        var artifacts Artifacts
        err := json.NewDecoder(resp.Body).Decode(&artifacts)
        if err != nil {
            fmt.Println("Error:", err)
        }
        setArtifactsCache(projectName, artifacts)
        // Parsing complete, check if artifacts already in pagerank. Rebuild if not (new project to the system)
        if !artifactsInCache(artifacts.Artifacts) {
            fetchPageRanks()
        }
        setArtifactsStatus(projectName, Status{Status: STATUS_COMPLETE})
        return true
    } else {
        setArtifactsStatus(projectName, Status{Status: STATUS_NOT_FOUND, Msg: "Cannot parse invalid project's dependency graph."})
    }
    return false
}
