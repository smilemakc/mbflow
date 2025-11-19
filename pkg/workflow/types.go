package workflow

type RetryPolicy struct {
	MaxAttempts int    `json:"max_attempts" yaml:"max_attempts"`
	Backoff     string `json:"backoff" yaml:"backoff"`
}

type NodeDef struct {
	ID        string         `json:"id" yaml:"id"`
	Type      string         `json:"type" yaml:"type"`
	Handler   string         `json:"handler" yaml:"handler"`
	Timeout   string         `json:"timeout" yaml:"timeout"`
	Retry     *RetryPolicy   `json:"retry" yaml:"retry"`
	Config    map[string]any `json:"config" yaml:"config"`
	Condition string         `json:"condition" yaml:"condition"`
}

type EdgeDef struct {
	From      string `json:"from" yaml:"from"`
	To        string `json:"to" yaml:"to"`
	Type      string `json:"type" yaml:"type"`
	Condition string `json:"condition" yaml:"condition"`
	Transform string `json:"transform" yaml:"transform"`
}

type TriggerDef struct {
	Type   string         `json:"type" yaml:"type"`
	ID     string         `json:"id" yaml:"id"`
	Config map[string]any `json:"config" yaml:"config"`
}

type Definition struct {
	Name        string       `json:"name" yaml:"name"`
	Version     string       `json:"version" yaml:"version"`
	Description string       `json:"description" yaml:"description"`
	Triggers    []TriggerDef `json:"triggers" yaml:"triggers"`
	Nodes       []NodeDef    `json:"nodes" yaml:"nodes"`
	Edges       []EdgeDef    `json:"edges" yaml:"edges"`
}
