package model

// AppsConfig maps to config/apps.json inside a product package
type AppsConfig struct {
	Project      ProjectConfig        `json:"project"`
	DeployConfig DeployConfigSettings `json:"deploy_config"`
	Apps         map[string]AppConfig `json:"apps"`
}

type ProjectConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type DeployConfigSettings struct {
	ParallelExecution  bool `json:"parallel_execution"`
	StopOnFailure      bool `json:"stop_on_failure"`
	BackupBeforeDeploy bool `json:"backup_before_deploy"`
}

type AppConfig struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Enabled         bool              `json:"enabled"`
	Dependencies    []string          `json:"dependencies"`
	DeployOrder     int               `json:"deploy_order"`
	HealthCheck     HealthCheckConfig `json:"health_check"`
	RollbackSupport bool              `json:"rollback_support"`
	Scripts         ScriptsConfig     `json:"scripts"`
}

type HealthCheckConfig struct {
	URL     string `json:"url"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type ScriptsConfig struct {
	PreDeploy   string `json:"pre_deploy"`
	Deploy      string `json:"deploy"`
	PostDeploy  string `json:"post_deploy"`
	HealthCheck string `json:"health_check"`
	Rollback    string `json:"rollback"`
}
