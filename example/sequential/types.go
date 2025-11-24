package sequential

// WorkflowInput is the initial input for the workflow
type WorkflowInput struct {
	Val1 int `json:"val1"`
	Val2 int `json:"val2"`
	Mult int `json:"mult"`
}

// Step 1: Add
type AddInput struct {
	A int `json:"val1"` // Map from WorkflowInput.Val1
	B int `json:"val2"` // Map from WorkflowInput.Val2
	M int `json:"mult"` // Map from WorkflowInput.Mult
}

type AddOutput struct {
	Value int `json:"value"` // Result
	Mult  int `json:"mult"`  // Pass through
}

// Step 2: Multiply
type MultiplyInput struct {
	Value  int `json:"value"`
	Factor int `json:"mult"`
}

type MultiplyOutput struct {
	Value int `json:"value"`
}

// Step 3: Format
type FormatInput struct {
	Number int `json:"value"`
}

type FormatOutput struct {
	Message string `json:"message"`
}
