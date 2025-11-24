package builder

import workflow "github.com/sicko7947/gorkflow"

// WorkflowOption is a functional option for configuring workflows
type WorkflowOption func(*workflow.Workflow)

// WithDescription sets the workflow description
func WithDescription(description string) WorkflowOption {
	return func(w *workflow.Workflow) {
		w.SetDescription(description)
	}
}

// WithVersion sets the workflow version
func WithVersion(version string) WorkflowOption {
	return func(w *workflow.Workflow) {
		w.SetVersion(version)
	}
}

// WithDefaultConfig sets the default execution config
func WithDefaultConfig(config workflow.ExecutionConfig) WorkflowOption {
	return func(w *workflow.Workflow) {
		w.SetConfig(config)
	}
}

// WithTags sets workflow tags
func WithTags(tags map[string]string) WorkflowOption {
	return func(w *workflow.Workflow) {
		w.SetTags(tags)
	}
}

// ApplyOptions applies a list of options to a workflow
func ApplyOptions(w *workflow.Workflow, opts ...WorkflowOption) {
	for _, opt := range opts {
		opt(w)
	}
}
