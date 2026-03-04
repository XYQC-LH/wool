package scheduler

import "fmt"

type NoAvailableProviderError struct {
	Operation string
	ModelID   string
}

func (e *NoAvailableProviderError) Error() string {
	if e == nil {
		return "no available providers"
	}
	if e.Operation == "" {
		return fmt.Sprintf("no available providers for model: %s", e.ModelID)
	}
	return fmt.Sprintf("no available providers for route: %s/%s", e.Operation, e.ModelID)
}

type ModelOperationNotSupportedError struct {
	Operation string
	ModelID   string
}

func (e *ModelOperationNotSupportedError) Error() string {
	if e == nil {
		return "model operation not supported"
	}
	if e.Operation == "" {
		return fmt.Sprintf("model %s does not support this operation", e.ModelID)
	}
	return fmt.Sprintf("model %s does not support operation: %s", e.ModelID, e.Operation)
}
