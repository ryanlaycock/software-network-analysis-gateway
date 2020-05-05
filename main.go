package main

import (
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "net/http"
    "os"
    "sync"
    "time"
)

const (
    STATUS_ERROR       = "error"
    STATUS_NOT_FOUND   = "not_found"
    STATUS_COMPLETE    = "complete"
    STATUS_IN_PROGRESS = "in_progress"
    STATUS_CANNOT_PARSE = "cannot_parse"
)

var (
    projectStatuses   ProjectStatuses
    projectCache      ProjectCache
    artifactsCache    ArtifactsCache
    pageRankCache     PageRankCache
    artifactsStatuses ArtifactsStatuses
    snaUrl            string
    dnaUrl            string
)

type Status struct {
    Status string `json:"status"`
    Msg    string `json:"msg"`
}

type ProjectCache struct {
    Mutex    sync.Mutex
    Projects map[string]Project
}

type ArtifactsCache struct {
    Mutex     sync.Mutex
    Artifacts map[string]Artifacts
}

type PageRankCache struct {
    Mutex sync.Mutex
    Ranks map[string]Ranks
}

type ProjectStatuses struct {
    Mutex  sync.Mutex
    Status map[string]Status
}

type ArtifactsStatuses struct {
    Mutex  sync.Mutex
    Status map[string]Status
}

type IncomingStatus struct {
    Status      string `json:"status"`
    Msg         string `json:"msg"`
    ProjectName string `json:"project_name"`
}

func updateProjectStatus(w http.ResponseWriter, r *http.Request) {
    var incomingStatus IncomingStatus
    err := json.NewDecoder(r.Body).Decode(&incomingStatus)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    fmt.Println(logTime(), "Progress update:", incomingStatus)
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

func setArtifactsStatus(projectName string, status Status) {
    artifactsStatuses.Mutex.Lock()
    artifactsStatuses.Status[projectName] = status
    artifactsStatuses.Mutex.Unlock()
}

func getArtifactsStatus(projectName string) (Status, bool) {
    projectStatuses.Mutex.Lock()
    status, found := artifactsStatuses.Status[projectName]
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

func setArtifactsCache(projectName string, artifacts Artifacts) {
    artifactsCache.Mutex.Lock()
    artifactsCache.Artifacts[projectName] = artifacts
    artifactsCache.Mutex.Unlock()
}

func getArtifactFromCache(projectName string) (Artifacts, bool) {
    artifactsCache.Mutex.Lock()
    artifacts, found := artifactsCache.Artifacts[projectName]
    artifactsCache.Mutex.Unlock()
    return artifacts, found
}

func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Add("Content-Type", "application/json")
        w.Header().Set("Access-Control-Allow-Origin", "*")
        next.ServeHTTP(w, r)
    })
}

func logTime() string {
    return time.Now().Format("2006-01-02 15:04:05")
}

func main() {
    fmt.Println(logTime(), "Initiating Software Network Analysis gateway.")
    snaUrl = os.Getenv("SNA_ADDR")
    dnaUrl = os.Getenv("DNA_ADDR")

    projectStatuses = ProjectStatuses{Status: map[string]Status{}, Mutex: sync.Mutex{}}
    artifactsStatuses = ArtifactsStatuses{Status: map[string]Status{}, Mutex: sync.Mutex{}}
    projectCache = ProjectCache{Projects: map[string]Project{}, Mutex: sync.Mutex{}}
    artifactsCache = ArtifactsCache{Artifacts: map[string]Artifacts{}, Mutex: sync.Mutex{}}
    pageRankCache = PageRankCache{Ranks: map[string]Ranks{}, Mutex: sync.Mutex{}}

    router := mux.NewRouter()
    router.Use(middleware)
    router.Use(mux.CORSMethodMiddleware(router))

    // External endpoints
    router.HandleFunc("/projects/{owner}/{repo}/metrics", getMetrics).Methods("GET")
    router.HandleFunc("/artifacts/{owner}/{repo}/metrics", getArtifacts).Methods("GET")

    // Internal endpoints
    router.HandleFunc("/projects/{owner}/{repo}/status", updateProjectStatus).Methods("POST")

    err := http.ListenAndServe(":8070", router)
    if err != nil {
        fmt.Println(logTime(), "An error occurred initiating server:", err)
    }
}
