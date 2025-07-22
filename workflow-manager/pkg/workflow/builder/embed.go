package builder

import (
	_ "embed"
)

var (
	//go:embed workflow.yaml
	workflowBuilder string
)
