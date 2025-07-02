package domain

type TaskContext struct {
	Task         *Task
	Project      *Project
	Dependencies []*Task
	Children     []*Task
	Related      []*Task
}

type SearchOptions struct {
	ProjectID *string
	Limit     int
	Offset    int
}

type SearchResult struct {
	Task      *Task   `json:"task"`
	Score     float64 `json:"score"`
	MatchType string  `json:"matchType"`
	Snippet   string  `json:"snippet"`
}

type NextTaskCriteria struct {
	ProjectID string
	Exclude   []string
}

type SufficiencyReport struct {
	TaskID     string `json:"taskId"`
	Sufficient bool   `json:"sufficient"`
	Missing    []string `json:"missing"`
	Stale      []string `json:"stale"`
}

type ContextRetriever interface {
	GetTaskContext(taskID string) (*TaskContext, error)
	Search(query string, opts SearchOptions) ([]*SearchResult, error)
	GetNextTask(criteria NextTaskCriteria) (*Task, error)
	CheckSufficiency(taskID string) (*SufficiencyReport, error)
}