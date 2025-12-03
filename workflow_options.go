package gorkflow

// WorkflowOption is a functional option for configuring workflows
// Defined in workflow.go, but helper functions are here

// WithDescription sets the workflow description
func WithDescription(description string) WorkflowOption {
	return func(w *Workflow) {
		w.SetDescription(description)
	}
}

// WithVersion sets the workflow version
func WithVersion(version string) WorkflowOption {
	return func(w *Workflow) {
		w.SetVersion(version)
	}
}

// WithDefaultConfig sets the default execution config
func WithDefaultConfig(config ExecutionConfig) WorkflowOption {
	return func(w *Workflow) {
		w.SetConfig(config)
	}
}

// WithWorkflowTags sets workflow tags
func WithWorkflowTags(tags map[string]string) WorkflowOption {
	return func(w *Workflow) {
		w.SetTags(tags)
	}
}

// ApplyOptions applies a list of options to a workflow
func ApplyOptions(w *Workflow, opts ...WorkflowOption) {
	for _, opt := range opts {
		opt(w)
	}
}
