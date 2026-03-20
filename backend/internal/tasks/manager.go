package tasks

import (
	"fmt"
	"sync"
	"time"

	"jav-shadcn/backend/internal/contracts"
)

type Manager struct {
	mu    sync.RWMutex
	tasks map[string]contracts.TaskDTO
}

func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]contracts.TaskDTO),
	}
}

func (m *Manager) Create(taskType string, metadata map[string]any) contracts.TaskDTO {
	now := nowUTC()
	task := contracts.TaskDTO{
		TaskID:    fmt.Sprintf("%s-%d", sanitizeTaskType(taskType), time.Now().UnixNano()),
		Type:      taskType,
		Status:    contracts.TaskPending,
		CreatedAt: now,
		Progress:  0,
		Metadata:  metadata,
	}

	m.mu.Lock()
	m.tasks[task.TaskID] = task
	m.mu.Unlock()

	return task
}

func (m *Manager) Start(taskID, message string) contracts.TaskDTO {
	return m.update(taskID, func(task contracts.TaskDTO) contracts.TaskDTO {
		task.Status = contracts.TaskRunning
		task.StartedAt = nowUTC()
		task.Message = message
		return task
	})
}

func (m *Manager) Progress(taskID string, progress int, message string) contracts.TaskDTO {
	return m.update(taskID, func(task contracts.TaskDTO) contracts.TaskDTO {
		task.Status = contracts.TaskRunning
		task.Progress = progress
		task.Message = message
		return task
	})
}

// ProgressWithMetadata updates running progress and shallow-merges patch into task.Metadata (for scan UI counters, etc.).
func (m *Manager) ProgressWithMetadata(taskID string, progress int, message string, patch map[string]any) contracts.TaskDTO {
	return m.update(taskID, func(task contracts.TaskDTO) contracts.TaskDTO {
		task.Status = contracts.TaskRunning
		task.Progress = progress
		task.Message = message
		if task.Metadata == nil {
			task.Metadata = map[string]any{}
		}
		for k, v := range patch {
			task.Metadata[k] = v
		}
		return task
	})
}

func (m *Manager) Complete(taskID, message string) contracts.TaskDTO {
	return m.update(taskID, func(task contracts.TaskDTO) contracts.TaskDTO {
		task.Status = contracts.TaskCompleted
		task.Progress = 100
		task.Message = message
		task.FinishedAt = nowUTC()
		return task
	})
}

func (m *Manager) Fail(taskID, code, message string) contracts.TaskDTO {
	return m.update(taskID, func(task contracts.TaskDTO) contracts.TaskDTO {
		task.Status = contracts.TaskFailed
		task.ErrorCode = code
		task.ErrorMessage = message
		task.FinishedAt = nowUTC()
		return task
	})
}

func (m *Manager) Get(taskID string) (contracts.TaskDTO, bool) {
	m.mu.RLock()
	task, ok := m.tasks[taskID]
	m.mu.RUnlock()
	return task, ok
}

func (m *Manager) update(taskID string, mutate func(contracts.TaskDTO) contracts.TaskDTO) contracts.TaskDTO {
	m.mu.Lock()
	defer m.mu.Unlock()

	task := m.tasks[taskID]
	task = mutate(task)
	m.tasks[taskID] = task
	return task
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func sanitizeTaskType(taskType string) string {
	sanitized := make([]rune, 0, len(taskType))
	for _, r := range taskType {
		if r == '.' || r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sanitized = append(sanitized, r)
			continue
		}
		sanitized = append(sanitized, '-')
	}
	return string(sanitized)
}
