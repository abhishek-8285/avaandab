package providers

import "sync"

var (
	mu       sync.RWMutex
	registry = map[string]TelematicsProvider{}
)

// Register adds a provider to the process-local registry. Registration is
// idempotent by provider name; later registrations overwrite earlier ones.
func Register(p TelematicsProvider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get returns the registered provider with the given name.
func Get(name string) (TelematicsProvider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns a snapshot of all registered providers keyed by name.
func All() map[string]TelematicsProvider {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]TelematicsProvider, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
