package service

import (
	"deploy-platform/model"
	"encoding/json"
	"os"
	"sync"
)

type HistoryService struct {
	mu       sync.Mutex
	filePath string
}

func NewHistoryService(filePath string) *HistoryService {
	return &HistoryService{filePath: filePath}
}

func (s *HistoryService) Save(task *model.DeployTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.load()
	found := false
	for i, t := range store.Tasks {
		if t.ID == task.ID {
			stored := *task
			if len(stored.Logs) > 2000 {
				stored.Logs = stored.Logs[len(stored.Logs)-2000:]
			}
			store.Tasks[i] = &stored
			found = true
			break
		}
	}
	if !found {
		stored := *task
		if len(stored.Logs) > 2000 {
			stored.Logs = stored.Logs[len(stored.Logs)-2000:]
		}
		store.Tasks = append([]*model.DeployTask{&stored}, store.Tasks...)
	}
	return s.write(store)
}

func (s *HistoryService) List() ([]*model.DeployTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.load()
	return store.Tasks, nil
}

func (s *HistoryService) Get(id string) (*model.DeployTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, _ := s.load()
	for _, t := range store.Tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *HistoryService) load() (*model.HistoryStore, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return &model.HistoryStore{}, nil
	}
	var store model.HistoryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return &model.HistoryStore{}, nil
	}
	return &store, nil
}

func (s *HistoryService) write(store *model.HistoryStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}
