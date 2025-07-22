package runtime

const (
	// WorkflowEngineDatabaseName mongo db database to store workflow automation information
	WorkflowEngineDatabaseName = "workflow-engine"
)

const (
	// BaseImageVersionCollection Base image version collection
	BaseImageVersionCollection = "base-image-versions"

	// ModulesCollection collection of modules
	ModulesCollection = "modules"

	// DummyTemplateCollection temporary runtime for templates
	DummyTemplateCollection = "dummy-templates"

	// DummyWorkflowCollection temporary runtime for workflow executions
	DummyWorkflowCollection = "dummy-workflows"
)
