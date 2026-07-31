package model

import "time"

// Package represents an uploaded product package
type Package struct {
	ID           string    `json:"id"`
	FileName     string    `json:"file_name"`
	SizeBytes    int64     `json:"size_bytes"`
	UploadedAt   time.Time `json:"uploaded_at"`
	ProjectName  string    `json:"project_name"`
	Version      string    `json:"version"`
	Environment  string    `json:"environment"`
	Apps         []AppInfo `json:"apps"`
	WorkspaceDir string    `json:"workspace_dir"`
}

// AppInfo is a simplified view for the frontend
type AppInfo struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Enabled         bool     `json:"enabled"`
	DeployOrder     int      `json:"deploy_order"`
	Dependencies    []string `json:"dependencies"`
	RollbackSupport bool     `json:"rollback_support"`
	HasHealthCheck  bool     `json:"has_health_check"`
}

// PackageStore persists packages to disk
type PackageStore struct {
	Packages []*Package `json:"packages"`
}
