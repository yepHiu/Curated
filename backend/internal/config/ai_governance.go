package config

import "fmt"

type AIGovernanceConfig struct {
	Enabled        bool   `json:"enabled"`
	ReadOnly       bool   `json:"readOnly"`
	Privacy        string `json:"privacy"`
	StepLimit      int    `json:"stepLimit"`
	WritePerMinute int    `json:"writePerMinute"`
	RetentionDays  int    `json:"retentionDays"`
}

func DefaultAIGovernance() AIGovernanceConfig {
	return AIGovernanceConfig{Privacy: "auto", StepLimit: 15, WritePerMinute: 10, RetentionDays: 30}
}
func (c AIGovernanceConfig) Validate() error {
	if c.Privacy != "auto" && c.Privacy != "minimal" {
		return fmt.Errorf("privacy must be auto or minimal")
	}
	if c.StepLimit < 1 || c.StepLimit > 30 {
		return fmt.Errorf("stepLimit must be between 1 and 30")
	}
	if c.WritePerMinute < 1 || c.WritePerMinute > 60 {
		return fmt.Errorf("writePerMinute must be between 1 and 60")
	}
	if c.RetentionDays < 7 || c.RetentionDays > 365 {
		return fmt.Errorf("retentionDays must be between 7 and 365")
	}
	return nil
}
