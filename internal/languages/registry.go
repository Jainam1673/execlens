package languages

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages the set of registered language adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]LanguageAdapter
	ordered  []string
}

// NewDefaultRegistry initializes the registry with the six standard language adapters.
func NewDefaultRegistry() *Registry {
	r := &Registry{
		adapters: make(map[string]LanguageAdapter),
	}
	r.Register(NewCPPAdapter())
	r.Register(NewRustAdapter())
	r.Register(NewGoAdapter())
	r.Register(NewJavaAdapter())
	r.Register(NewPythonAdapter())
	r.Register(NewJSAdapter())
	return r
}

// Register adds an adapter to the registry.
func (r *Registry) Register(adapter LanguageAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := adapter.ID()
	if _, exists := r.adapters[id]; !exists {
		r.ordered = append(r.ordered, id)
	}
	r.adapters[id] = adapter
}

// Get returns the adapter for a given language ID.
func (r *Registry) Get(id string) (LanguageAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, exists := r.adapters[id]
	if !exists {
		return nil, fmt.Errorf("language adapter '%s' not registered", id)
	}
	return adapter, nil
}

// List returns all registered adapters in registration order.
func (r *Registry) List() []LanguageAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]LanguageAdapter, 0, len(r.ordered))
	for _, id := range r.ordered {
		res = append(res, r.adapters[id])
	}
	return res
}

// DetectAll probes the host environment for all registered languages.
func (r *Registry) DetectAll(ctx context.Context) []RuntimeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var wg sync.WaitGroup
	results := make([]RuntimeInfo, len(r.ordered))

	for i, id := range r.ordered {
		wg.Add(1)
		go func(idx int, adapter LanguageAdapter) {
			defer wg.Done()
			isAvail, ver, missing := adapter.DetectAvailability(ctx)
			results[idx] = RuntimeInfo{
				ID:               adapter.ID(),
				DisplayName:      adapter.DisplayName(),
				IsAvailable:      isAvail,
				Version:          ver,
				MissingReason:    missing,
				RequiredTools:    adapter.RequiredTools(),
				IsCompiled:       adapter.IsCompiled(),
				DefaultWarmup:    adapter.DefaultWarmupIterations(),
				WarmupPolicyDesc: adapter.WarmupPolicyDescription(),
			}
		}(i, r.adapters[id])
	}

	wg.Wait()
	return results
}
