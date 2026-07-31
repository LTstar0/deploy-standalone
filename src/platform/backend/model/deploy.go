package model

import "time"

// DeployTask represents one deployment execution
type DeployTask struct {
	ID           string     `json:"id"`
	PackageID    string     `json:"package_id"`
	PackageName  string     `json:"package_name"`
	Version      string     `json:"version"`
	Environment  string     `json:"environment"`
	SelectedApps []string   `json:"selected_apps"`
	Status       string     `json:"status"` // running, success, failed, rolling_back, rolled_back
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Logs         []LogLine  `json:"logs"`
	DeployedApps []string   `json:"deployed_apps"`
	FailedApp    string     `json:"failed_app,omitempty"`
}

type LogLine struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"` // info, success, error, warn, step
	Message string    `json:"message"`
}

type WSMessage struct {
	Type string      `json:"type"` // log, status, done
	Data interface{} `json:"data"`
}

type StartDeployRequest struct {
	PackageID    string   `json:"package_id" binding:"required"`
	SelectedApps []string `json:"selected_apps" binding:"required"`
}

type HistoryStore struct {
	Tasks []*DeployTask `json:"tasks"`
}
