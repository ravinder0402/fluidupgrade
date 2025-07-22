package builder

import (
	_ "embed"
)

var (
	//go:embed builder.yaml
	moduleBuilder string

	//go:embed dockerfile.tmpl
	dockerfile string
)
