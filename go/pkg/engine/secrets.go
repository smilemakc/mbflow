package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// RedactedValue replaces secret values in persisted and emitted data.
const RedactedValue = "***"

// MinSecretLength is the shortest value a SecretRedactor redacts; shorter
// values would match too much unrelated text.
const MinSecretLength = 4

// SecretRedactor holds the secret values of one execution (e.g. decrypted
// credentials) and replaces them with RedactedValue in copies of data that is
// persisted or emitted (node configs, inputs, outputs, errors, events).
//
// Redaction never modifies its argument: executors and downstream nodes keep
// working with the real values. All methods are safe for concurrent use and
// on a nil receiver, which redacts nothing.
type SecretRedactor struct {
	mu       sync.RWMutex
	secrets  map[string]struct{}
	replacer *strings.Replacer
}

// NewSecretRedactor returns a redactor for the given secret values.
func NewSecretRedactor(values ...string) *SecretRedactor {
	r := &SecretRedactor{secrets: make(map[string]struct{})}
	r.Add(values...)
	return r
}

// Add registers secret values. Values shorter than MinSecretLength are ignored.
func (r *SecretRedactor) Add(values ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	changed := false
	for _, v := range values {
		if len(v) < MinSecretLength {
			continue
		}
		if _, ok := r.secrets[v]; ok {
			continue
		}
		r.secrets[v] = struct{}{}
		changed = true
	}
	if !changed {
		return
	}
	// Longest first, so a secret containing another one is replaced whole.
	list := make([]string, 0, len(r.secrets))
	for s := range r.secrets {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		if len(list[i]) != len(list[j]) {
			return len(list[i]) > len(list[j])
		}
		return list[i] < list[j]
	})
	pairs := make([]string, 0, 2*len(list))
	for _, s := range list {
		pairs = append(pairs, s, RedactedValue)
	}
	r.replacer = strings.NewReplacer(pairs...)
}

// Len returns the number of registered secrets.
func (r *SecretRedactor) Len() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.secrets)
}

func (r *SecretRedactor) getReplacer() *strings.Replacer {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.replacer
}

// String returns s with every secret replaced by RedactedValue.
func (r *SecretRedactor) String(s string) string {
	rep := r.getReplacer()
	if rep == nil || s == "" {
		return s
	}
	return rep.Replace(s)
}

// Value returns a redacted deep copy of v. Maps, slices and strings are
// copied with secrets replaced; scalars are returned unchanged; other values
// (structs, typed maps) are converted to their JSON form first.
func (r *SecretRedactor) Value(v any) any {
	rep := r.getReplacer()
	if rep == nil {
		return v
	}
	return redactValue(rep, v)
}

// Map returns a redacted deep copy of m (nil stays nil).
func (r *SecretRedactor) Map(m map[string]any) map[string]any {
	rep := r.getReplacer()
	if rep == nil || m == nil {
		return m
	}
	return redactMap(rep, m)
}

// Error returns err with secrets removed from its message. The result still
// unwraps to err, so errors.Is/As keep working.
func (r *SecretRedactor) Error(err error) error {
	rep := r.getReplacer()
	if rep == nil || err == nil {
		return err
	}
	msg := err.Error()
	redacted := rep.Replace(msg)
	if redacted == msg {
		return err
	}
	return &redactedError{msg: redacted, err: err}
}

// Event returns a copy of event with secrets removed from its error, output,
// message, input and variables.
func (r *SecretRedactor) Event(event ExecutionEvent) ExecutionEvent {
	rep := r.getReplacer()
	if rep == nil {
		return event
	}
	event.Error = r.Error(event.Error)
	if event.Output != nil {
		event.Output = redactValue(rep, event.Output)
	}
	event.Message = rep.Replace(event.Message)
	if event.Input != nil {
		event.Input = redactMap(rep, event.Input)
	}
	if event.Variables != nil {
		event.Variables = redactMap(rep, event.Variables)
	}
	return event
}

// redactedError carries a redacted message while still unwrapping to the
// original error for errors.Is/As.
type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.err }

func redactMap(rep *strings.Replacer, m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[rep.Replace(k)] = redactValue(rep, v)
	}
	return out
}

func redactValue(rep *strings.Replacer, v any) any {
	switch val := v.(type) {
	case nil:
		return nil
	case string:
		return rep.Replace(val)
	case map[string]any:
		return redactMap(rep, val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = redactValue(rep, item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(val))
		for k, s := range val {
			out[rep.Replace(k)] = rep.Replace(s)
		}
		return out
	case []string:
		out := make([]string, len(val))
		for i, s := range val {
			out[i] = rep.Replace(s)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(val))
		for i, m := range val {
			out[i] = redactMap(rep, m)
		}
		return out
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64, json.Number:
		return val
	case error:
		return rep.Replace(val.Error())
	default:
		return redactViaJSON(rep, val)
	}
}

// redactViaJSON converts an arbitrary value to its generic JSON form and
// redacts that; values that cannot be marshaled are redacted as text.
func redactViaJSON(rep *strings.Replacer, v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		return rep.Replace(fmt.Sprint(v))
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return rep.Replace(string(data))
	}
	return redactValue(rep, generic)
}
