package service

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"deploy-platform/model"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PackageService struct {
	mu          sync.Mutex
	dataDir     string
	packageFile string
}

func NewPackageService(dataDir string) *PackageService {
	pkgDir := filepath.Join(dataDir, "packages")
	wsDir := filepath.Join(dataDir, "workspaces")
	os.MkdirAll(pkgDir, 0755)
	os.MkdirAll(wsDir, 0755)
	return &PackageService{
		dataDir:     dataDir,
		packageFile: filepath.Join(dataDir, "packages.json"),
	}
}

func (s *PackageService) Upload(fileName string, fileData io.Reader, size int64) (*model.Package, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()[:8]
	pkgDir := filepath.Join(s.dataDir, "packages")
	savePath := filepath.Join(pkgDir, id+"_"+fileName)

	f, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	written, err := io.Copy(f, fileData)
	f.Close()
	if err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("save file: %w", err)
	}

	wsDir := filepath.Join(s.dataDir, "workspaces", id)
	if err := s.extract(savePath, wsDir); err != nil {
		os.Remove(savePath)
		return nil, fmt.Errorf("extract: %w", err)
	}

	config, err := s.readAppsConfig(wsDir)
	if err != nil {
		os.Remove(savePath)
		os.RemoveAll(wsDir)
		return nil, fmt.Errorf("read apps.json: %w", err)
	}

	var apps []model.AppInfo
	for key, app := range config.Apps {
		apps = append(apps, model.AppInfo{
			Key:             key,
			Name:            app.Name,
			Description:     app.Description,
			Enabled:         app.Enabled,
			DeployOrder:     app.DeployOrder,
			Dependencies:    app.Dependencies,
			RollbackSupport: app.RollbackSupport,
			HasHealthCheck:  app.HealthCheck.URL != "" || app.HealthCheck.Command != "" || app.Scripts.HealthCheck != "",
		})
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].DeployOrder < apps[j].DeployOrder })

	pkg := &model.Package{
		ID:           id,
		FileName:     fileName,
		SizeBytes:    written,
		UploadedAt:   timeNow(),
		ProjectName:  config.Project.Name,
		Version:      config.Project.Version,
		Environment:  config.Project.Environment,
		Apps:         apps,
		WorkspaceDir: wsDir,
	}

	store, _ := s.loadStore()
	store.Packages = append([]*model.Package{pkg}, store.Packages...)
	s.writeStore(store)

	return pkg, nil
}

func (s *PackageService) List() []*model.Package {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.loadStore()
	return store.Packages
}

func (s *PackageService) Get(id string) (*model.Package, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.loadStore()
	for _, p := range store.Packages {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *PackageService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.loadStore()
	var updated []*model.Package
	var found *model.Package
	for _, p := range store.Packages {
		if p.ID == id {
			found = p
		} else {
			updated = append(updated, p)
		}
	}
	if found == nil {
		return os.ErrNotExist
	}
	// clean files
	pkgDir := filepath.Join(s.dataDir, "packages")
	entries, _ := os.ReadDir(pkgDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id+"_") {
			os.Remove(filepath.Join(pkgDir, e.Name()))
		}
	}
	os.RemoveAll(found.WorkspaceDir)

	store.Packages = updated
	return s.writeStore(store)
}

func (s *PackageService) GetAppsConfig(id string) (*model.AppsConfig, error) {
	pkg, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return s.readAppsConfig(pkg.WorkspaceDir)
}

// extract handles .tar.gz and .zip
func (s *PackageService) extract(archivePath, destDir string) error {
	os.MkdirAll(destDir, 0755)
	if strings.HasSuffix(archivePath, ".zip") {
		return s.extractZip(archivePath, destDir)
	}
	return s.extractTarGz(archivePath, destDir)
}

func (s *PackageService) extractTarGz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)) {
			continue // zip slip protection
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
		}
	}
	return nil
}

func (s *PackageService) extractZip(path, dest string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

// readAppsConfig finds config/apps.json in workspace (may be nested one level)
func (s *PackageService) readAppsConfig(wsDir string) (*model.AppsConfig, error) {
	candidates := []string{
		filepath.Join(wsDir, "config", "apps.json"),
	}
	// check one level deep (e.g., extracted folder name)
	entries, _ := os.ReadDir(wsDir)
	for _, e := range entries {
		if e.IsDir() {
			candidates = append(candidates, filepath.Join(wsDir, e.Name(), "config", "apps.json"))
		}
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var config model.AppsConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("parse apps.json: %w", err)
		}
		return &config, nil
	}
	return nil, fmt.Errorf("config/apps.json not found in package")
}

func (s *PackageService) loadStore() (*model.PackageStore, error) {
	data, err := os.ReadFile(s.packageFile)
	if err != nil {
		return &model.PackageStore{}, nil
	}
	var store model.PackageStore
	json.Unmarshal(data, &store)
	return &store, nil
}

func (s *PackageService) writeStore(store *model.PackageStore) error {
	data, _ := json.MarshalIndent(store, "", "  ")
	return os.WriteFile(s.packageFile, data, 0644)
}

// timeNow is extracted for testability
var timeNow = time.Now
