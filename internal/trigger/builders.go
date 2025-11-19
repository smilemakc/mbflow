package trigger

type HTTPTriggerBuilder struct {
	cfg HTTPConfig
}

func NewHTTPTriggerBuilder() *HTTPTriggerBuilder                  { return &HTTPTriggerBuilder{} }
func (b *HTTPTriggerBuilder) Path(p string) *HTTPTriggerBuilder   { b.cfg.Path = p; return b }
func (b *HTTPTriggerBuilder) Method(m string) *HTTPTriggerBuilder { b.cfg.Method = m; return b }
func (b *HTTPTriggerBuilder) Build() *HTTPTrigger                 { return NewHTTP(b.cfg) }

type ManualTriggerBuilder struct{}

func NewManualTriggerBuilder() *ManualTriggerBuilder  { return &ManualTriggerBuilder{} }
func (b *ManualTriggerBuilder) Build() *ManualTrigger { return NewManual() }
