package events

// FunctionBuildProgressEvent represents a build progress event for the SST live console
type FunctionBuildProgressEvent struct {
	FunctionID string `json:"functionID"`
	Stage      string `json:"stage"`
	Message    string `json:"message"`
}
