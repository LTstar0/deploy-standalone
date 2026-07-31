package service

import (
	"bufio"
	"context"
	"deploy-platform/model"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeployService orchestrates shell script execution
type DeployService struct {
	mu         sync.Mutex
	packageSvc *PackageService
	historySvc *HistoryService
	activeTasks map[string]*DeployTask
}

type DeployTask struct {
	task     *model.DeployTask
	cancel   context.CancelFunc
	listeners map[string]chan model.WSMessage
	mu        sync.Mutex
}

func NewDeployService(pkgSvc *PackageService, histSvc *HistoryService) *DeployService {
	return &DeployService{
		packageSvc:  pkgSvc,
		historySvc:  histSvc,
		activeTasks: make(map[string]*DeployTask),
	}
}

// Subscribe returns a channel to receive WSMessages for a task
func (s *DeployService) Subscribe(taskID, subID string) (chan model.WSMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt, ok := s.activeTasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %s not found or already finished", taskID)
	}
	ch := make(chan model.WSMessage, 100)
	dt.mu.Lock()
	dt.listeners[subID] = ch
	dt.mu.Unlock()

	// Send existing logs as history
	go func() {
		dt.mu.Lock()
		logs := make([]model.LogLine, len(dt.task.Logs))
		copy(logs, dt.task.Logs)
		dt.mu.Unlock()
		for _, l := range logs {
			ch <- model.WSMessage{Type: "log", Data: l}
		}
	}()
	return ch, nil
}

func (s *DeployService) Unsubscribe(taskID, subID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt, ok := s.activeTasks[taskID]
	if !ok {
		return
	}
	dt.mu.Lock()
	if ch, ok := dt.listeners[subID]; ok {
		close(ch)
		delete(dt.listeners, subID)
	}
	dt.mu.Unlock()
}

// StartDeploy begins the deployment asynchronously
func (s *DeployService) StartDeploy(req model.StartDeployRequest) (*model.DeployTask, error) {
	if s.HasActiveTask() {
		return nil, fmt.Errorf("已有其他部署或回滚任务正在执行")
	}

	pkg, err := s.packageSvc.Get(req.PackageID)
	if err != nil {
		return nil, fmt.Errorf("package not found: %w", err)
	}

	configPath := s.findConfigPath(pkg.WorkspaceDir)
	if configPath == "" {
		return nil, fmt.Errorf("config/apps.json not found")
	}
	data, _ := os.ReadFile(configPath)
	var config model.AppsConfig
	json.Unmarshal(data, &config)

	// Resolve dependencies
	expanded := s.resolveDeps(req.SelectedApps, config.Apps)
	sorted := s.sortByOrder(expanded, config.Apps)

	taskID := uuid.New().String()[:8]
	now := time.Now()
	task := &model.DeployTask{
		ID:           taskID,
		PackageID:    req.PackageID,
		PackageName:  pkg.ProjectName,
		Version:      pkg.Version,
		Environment:  pkg.Environment,
		SelectedApps: sorted,
		Status:       "running",
		StartedAt:    now,
		Logs:         []model.LogLine{},
		DeployedApps: []string{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	dt := &DeployTask{
		task:      task,
		cancel:    cancel,
		listeners: make(map[string]chan model.WSMessage),
	}

	s.mu.Lock()
	s.activeTasks[taskID] = dt
	s.mu.Unlock()

	s.historySvc.Save(task)

	go s.runDeploy(ctx, dt, config, pkg.WorkspaceDir)

	return task, nil
}

// HasActiveTask returns true if there is a running or rolling back task
func (s *DeployService) HasActiveTask() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, dt := range s.activeTasks {
		dt.mu.Lock()
		status := dt.task.Status
		dt.mu.Unlock()
		if status == "running" || status == "rolling_back" {
			return true
		}
	}
	return false
}

func (s *DeployService) GetActiveTask(id string) *model.DeployTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt, ok := s.activeTasks[id]
	if !ok {
		return nil
	}
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.task
}

// runDeploy is the main deploy orchestration loop
func (s *DeployService) runDeploy(ctx context.Context, dt *DeployTask, config model.AppsConfig, wsDir string) {
	stopOnFailure := config.DeployConfig.StopOnFailure
	baseDir := s.findBaseDir(wsDir)

	s.emit(dt, "step", fmt.Sprintf("🚀 开始部署 — %s v%s (%s)", dt.task.PackageName, dt.task.Version, dt.task.Environment))
	s.emit(dt, "info", fmt.Sprintf("共 %d 个应用将按顺序部署", len(dt.task.SelectedApps)))

	for _, appKey := range dt.task.SelectedApps {
		select {
		case <-ctx.Done():
			s.emit(dt, "warn", "部署被取消")
			s.finish(dt, "failed")
			return
		default:
		}

		appCfg := config.Apps[appKey]
		s.emit(dt, "step", fmt.Sprintf("▶ 部署应用: %s (%s)", appCfg.Name, appKey))

		// Pre-deploy
		if appCfg.Scripts.PreDeploy != "" {
			s.emit(dt, "info", "执行 pre-deploy...")
			if err := s.runScript(ctx, dt, baseDir, appCfg.Scripts.PreDeploy); err != nil {
				s.emit(dt, "error", fmt.Sprintf("pre-deploy 失败: %v", err))
				dt.mu.Lock()
				dt.task.FailedApp = appKey
				dt.mu.Unlock()
				if stopOnFailure {
					s.finish(dt, "failed")
					return
				}
				continue
			}
		}

		// Deploy
		if appCfg.Scripts.Deploy != "" {
			s.emit(dt, "info", "执行 deploy...")
			if err := s.runScript(ctx, dt, baseDir, appCfg.Scripts.Deploy); err != nil {
				s.emit(dt, "error", fmt.Sprintf("deploy 失败: %v", err))
				dt.mu.Lock()
				dt.task.FailedApp = appKey
				dt.mu.Unlock()
				if stopOnFailure {
					s.finish(dt, "failed")
					return
				}
				continue
			}
		} else {
			s.emit(dt, "warn", "未配置 deploy 脚本，跳过")
		}

		// Health check
		if err := s.doHealthCheck(ctx, dt, baseDir, appCfg); err != nil {
			s.emit(dt, "error", fmt.Sprintf("健康检查失败: %v", err))
			dt.mu.Lock()
			dt.task.FailedApp = appKey
			dt.mu.Unlock()
			if stopOnFailure {
				s.finish(dt, "failed")
				return
			}
			continue
		}

		// Post-deploy
		if appCfg.Scripts.PostDeploy != "" {
			s.emit(dt, "info", "执行 post-deploy...")
			if err := s.runScript(ctx, dt, baseDir, appCfg.Scripts.PostDeploy); err != nil {
				s.emit(dt, "warn", fmt.Sprintf("post-deploy 执行失败（不阻塞）: %v", err))
			}
		}

		dt.mu.Lock()
		dt.task.DeployedApps = append(dt.task.DeployedApps, appKey)
		dt.mu.Unlock()
		s.emit(dt, "success", fmt.Sprintf("✔ %s 部署成功", appCfg.Name))
		s.historySvc.Save(dt.task)
	}

	if dt.task.FailedApp != "" {
		s.finish(dt, "failed")
	} else {
		s.emit(dt, "success", "🎉 所有应用部署成功！")
		s.finish(dt, "success")
	}
}

func (s *DeployService) doHealthCheck(ctx context.Context, dt *DeployTask, baseDir string, app model.AppConfig) error {
	// Priority: script > url > command
	if app.Scripts.HealthCheck != "" {
		s.emit(dt, "info", fmt.Sprintf("健康检查（脚本）: %s", app.Scripts.HealthCheck))
		return s.runScript(ctx, dt, baseDir, app.Scripts.HealthCheck)
	}

	timeout := app.HealthCheck.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	if app.HealthCheck.URL != "" {
		s.emit(dt, "info", fmt.Sprintf("健康检查（URL）: %s 超时: %ds", app.HealthCheck.URL, timeout))
		deadline := time.Now().Add(time.Duration(timeout) * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return fmt.Errorf("cancelled")
			default:
			}
			resp, err := http.Get(app.HealthCheck.URL)
			if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				resp.Body.Close()
				s.emit(dt, "success", "健康检查通过")
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
			s.emit(dt, "info", "  等待服务就绪...")
			time.Sleep(3 * time.Second)
		}
		return fmt.Errorf("URL 健康检查超时 (%ds)", timeout)
	}

	if app.HealthCheck.Command != "" {
		s.emit(dt, "info", fmt.Sprintf("健康检查（命令）: %s 超时: %ds", app.HealthCheck.Command, timeout))
		deadline := time.Now().Add(time.Duration(timeout) * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return fmt.Errorf("cancelled")
			default:
			}
			cmd := exec.Command("bash", "-c", app.HealthCheck.Command)
			cmd.Dir = baseDir
			if err := cmd.Run(); err == nil {
				s.emit(dt, "success", "健康检查通过")
				return nil
			}
			s.emit(dt, "info", "  等待服务就绪...")
			time.Sleep(3 * time.Second)
		}
		return fmt.Errorf("命令健康检查超时 (%ds)", timeout)
	}

	s.emit(dt, "info", "无健康检查配置，跳过")
	return nil
}

func (s *DeployService) runScript(ctx context.Context, dt *DeployTask, baseDir, scriptPath string) error {
	fullPath, err := filepath.Abs(filepath.Join(baseDir, scriptPath))
	if err != nil {
		return fmt.Errorf("解析脚本绝对路径失败: %w", err)
	}
	if _, err := os.Stat(fullPath); err != nil {
		return fmt.Errorf("脚本不存在: %s", scriptPath)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		absBaseDir = baseDir
	}

	cmd := exec.CommandContext(ctx, "bash", fullPath)
	cmd.Dir = absBaseDir


	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动脚本失败: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		s.emit(dt, "info", "  "+scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("脚本执行失败: %w", err)
	}
	return nil
}

func (s *DeployService) emit(dt *DeployTask, level, msg string) {
	line := model.LogLine{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	}
	dt.mu.Lock()
	dt.task.Logs = append(dt.task.Logs, line)
	// Copy listeners
	listeners := make(map[string]chan model.WSMessage, len(dt.listeners))
	for k, v := range dt.listeners {
		listeners[k] = v
	}
	dt.mu.Unlock()

	wsMsg := model.WSMessage{Type: "log", Data: line}
	for _, ch := range listeners {
		select {
		case ch <- wsMsg:
		default:
		}
	}
}

func (s *DeployService) finish(dt *DeployTask, status string) {
	now := time.Now()
	dt.mu.Lock()
	dt.task.Status = status
	dt.task.FinishedAt = &now
	dt.mu.Unlock()

	s.historySvc.Save(dt.task)

	// Notify done
	dt.mu.Lock()
	for _, ch := range dt.listeners {
		select {
		case ch <- model.WSMessage{Type: "done", Data: status}:
		default:
		}
	}
	dt.mu.Unlock()

	// Clean up active
	time.AfterFunc(5*time.Second, func() {
		s.mu.Lock()
		delete(s.activeTasks, dt.task.ID)
		s.mu.Unlock()
	})
}

// Rollback triggers rollback on deployed apps in reverse order
func (s *DeployService) Rollback(historyID string) (*model.DeployTask, error) {
	if s.HasActiveTask() {
		return nil, fmt.Errorf("已有其他部署或回滚任务正在执行")
	}

	hist, err := s.historySvc.Get(historyID)

	if err != nil {
		return nil, fmt.Errorf("history not found: %w", err)
	}
	if len(hist.DeployedApps) == 0 {
		return nil, fmt.Errorf("no deployed apps to rollback")
	}

	pkg, err := s.packageSvc.Get(hist.PackageID)
	if err != nil {
		return nil, fmt.Errorf("package not found: %w", err)
	}

	configPath := s.findConfigPath(pkg.WorkspaceDir)
	data, _ := os.ReadFile(configPath)
	var config model.AppsConfig
	json.Unmarshal(data, &config)

	// Reverse deployed apps
	reversed := make([]string, len(hist.DeployedApps))
	for i, key := range hist.DeployedApps {
		reversed[len(hist.DeployedApps)-1-i] = key
	}

	taskID := uuid.New().String()[:8]
	now := time.Now()
	task := &model.DeployTask{
		ID:           taskID,
		PackageID:    hist.PackageID,
		PackageName:  hist.PackageName,
		Version:      hist.Version,
		Environment:  hist.Environment,
		SelectedApps: reversed,
		Status:       "rolling_back",
		StartedAt:    now,
		Logs:         []model.LogLine{},
		DeployedApps: []string{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	dt := &DeployTask{
		task:      task,
		cancel:    cancel,
		listeners: make(map[string]chan model.WSMessage),
	}

	s.mu.Lock()
	s.activeTasks[taskID] = dt
	s.mu.Unlock()
	s.historySvc.Save(task)

	go s.runRollback(ctx, dt, config, pkg.WorkspaceDir, reversed)

	return task, nil
}

func (s *DeployService) runRollback(ctx context.Context, dt *DeployTask, config model.AppsConfig, wsDir string, apps []string) {
	baseDir := s.findBaseDir(wsDir)
	s.emit(dt, "step", "⏪ 开始回滚...")

	for _, appKey := range apps {
		appCfg, ok := config.Apps[appKey]
		if !ok {
			continue
		}
		s.emit(dt, "info", fmt.Sprintf("回滚: %s (%s)", appCfg.Name, appKey))
		if !appCfg.RollbackSupport || appCfg.Scripts.Rollback == "" {
			s.emit(dt, "warn", "  无回滚脚本，跳过")
			continue
		}
		if err := s.runScript(ctx, dt, baseDir, appCfg.Scripts.Rollback); err != nil {
			s.emit(dt, "error", fmt.Sprintf("  回滚失败: %v", err))
		} else {
			s.emit(dt, "success", fmt.Sprintf("  ✔ 回滚成功: %s", appKey))
			dt.mu.Lock()
			dt.task.DeployedApps = append(dt.task.DeployedApps, appKey)
			dt.mu.Unlock()
		}
	}

	s.emit(dt, "success", "回滚流程完成")
	s.finish(dt, "rolled_back")
}

// resolveDeps recursively adds missing dependencies
func (s *DeployService) resolveDeps(selected []string, apps map[string]model.AppConfig) []string {
	visited := make(map[string]bool)
	var visit func(key string)
	visit = func(key string) {
		if visited[key] {
			return
		}
		visited[key] = true
		if app, ok := apps[key]; ok {
			for _, dep := range app.Dependencies {
				visit(dep)
			}
		}
	}
	for _, k := range selected {
		visit(k)
	}
	result := make([]string, 0, len(visited))
	for k := range visited {
		result = append(result, k)
	}
	return result
}

func (s *DeployService) sortByOrder(keys []string, apps map[string]model.AppConfig) []string {
	sort.Slice(keys, func(i, j int) bool {
		return apps[keys[i]].DeployOrder < apps[keys[j]].DeployOrder
	})
	return keys
}

func (s *DeployService) findConfigPath(wsDir string) string {
	p := filepath.Join(wsDir, "config", "apps.json")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	entries, _ := os.ReadDir(wsDir)
	for _, e := range entries {
		if e.IsDir() {
			p2 := filepath.Join(wsDir, e.Name(), "config", "apps.json")
			if _, err := os.Stat(p2); err == nil {
				return p2
			}
		}
	}
	return ""
}

func (s *DeployService) findBaseDir(wsDir string) string {
	if _, err := os.Stat(filepath.Join(wsDir, "config", "apps.json")); err == nil {
		return wsDir
	}
	entries, _ := os.ReadDir(wsDir)
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(wsDir, e.Name(), "config", "apps.json")
			if _, err := os.Stat(p); err == nil {
				return filepath.Join(wsDir, e.Name())
			}
		}
	}
	return wsDir
}
