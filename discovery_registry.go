package endpoint

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// DiscoveryProvider discovers endpoint candidates for a request.
type DiscoveryProvider interface {
	Spec() ProviderSpec
	Discover(context.Context, DiscoveryRequest) (DiscoveryResult, error)
}

// DiscoveryRegistry stores endpoint discovery providers and their last-run status.
type DiscoveryRegistry struct {
	mu        sync.RWMutex
	providers []DiscoveryProvider
	status    map[string]ProviderStatus
}

// NewDiscoveryRegistry returns an empty discovery provider registry.
func NewDiscoveryRegistry() *DiscoveryRegistry {
	return &DiscoveryRegistry{status: map[string]ProviderStatus{}}
}

// Register appends provider unless another provider with the same name exists.
func (r *DiscoveryRegistry) Register(provider DiscoveryProvider) error {
	if r == nil {
		return fmt.Errorf("discovery: registry is nil")
	}
	if provider == nil {
		return fmt.Errorf("discovery: provider is nil")
	}
	spec := provider.Spec()
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("discovery: provider name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.providers {
		if existing.Spec().Name == spec.Name {
			return nil
		}
	}
	r.providers = append(r.providers, provider)
	if r.status == nil {
		r.status = map[string]ProviderStatus{}
	}
	r.status[spec.Name] = ProviderStatus{Spec: cloneProviderSpec(spec)}
	return nil
}

// Providers returns registered provider specs.
func (r *DiscoveryRegistry) Providers() []ProviderSpec {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderSpec, 0, len(r.providers))
	for _, provider := range r.providers {
		out = append(out, cloneProviderSpec(provider.Spec()))
	}
	return out
}

// Status returns current provider status.
func (r *DiscoveryRegistry) Status() []ProviderStatus {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderStatus, 0, len(r.status))
	for _, status := range r.status {
		status.Spec = cloneProviderSpec(status.Spec)
		out = append(out, status)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spec.Name < out[j].Spec.Name })
	return out
}

// Discover asks matching providers for endpoint candidates.
func (r *DiscoveryRegistry) Discover(ctx context.Context, req DiscoveryRequest) (DiscoveryResult, error) {
	if r == nil {
		return DiscoveryResult{}, fmt.Errorf("discovery: registry is nil")
	}
	providers := r.matchingProviders(req.Product)
	if len(req.Providers) > 0 {
		allowed := map[string]bool{}
		for _, name := range req.Providers {
			name = strings.TrimSpace(name)
			if name != "" {
				allowed[name] = true
			}
		}
		filtered := providers[:0]
		for _, provider := range providers {
			if allowed[provider.Spec().Name] {
				filtered = append(filtered, provider)
			}
		}
		providers = filtered
	}
	var out DiscoveryResult
	var firstErr error
	for _, provider := range providers {
		result, err := provider.Discover(ctx, req)
		r.recordRun(provider.Spec(), len(result.Candidates), err)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		out.EndpointRefs = append(out.EndpointRefs, result.EndpointRefs...)
		out.Candidates = append(out.Candidates, result.Candidates...)
		out.Probes = append(out.Probes, result.Probes...)
	}
	sort.SliceStable(out.Candidates, func(i, j int) bool {
		if out.Candidates[i].Score == out.Candidates[j].Score {
			return out.Candidates[i].ID < out.Candidates[j].ID
		}
		return out.Candidates[i].Score > out.Candidates[j].Score
	})
	if req.Limit > 0 && len(out.Candidates) > req.Limit {
		out.Candidates = out.Candidates[:req.Limit]
	}
	if len(out.Candidates) == 0 && firstErr != nil {
		return out, firstErr
	}
	return out, nil
}

func (r *DiscoveryRegistry) matchingProviders(product string) []DiscoveryProvider {
	product = strings.TrimSpace(product)
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []DiscoveryProvider
	for _, provider := range r.providers {
		spec := provider.Spec()
		if product == "" || SupportsProduct(spec, product) {
			out = append(out, provider)
		}
	}
	return out
}

func (r *DiscoveryRegistry) recordRun(spec ProviderSpec, results int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == nil {
		r.status = map[string]ProviderStatus{}
	}
	status := r.status[spec.Name]
	status.Spec = cloneProviderSpec(spec)
	status.LastRun = time.Now().UTC().Format(time.RFC3339)
	status.LastResults = results
	status.LastError = ""
	if err != nil {
		status.LastError = err.Error()
	}
	r.status[spec.Name] = status
}

// SupportsProduct reports whether provider spec declares product.
func SupportsProduct(spec ProviderSpec, product string) bool {
	for _, candidate := range spec.Products {
		if strings.TrimSpace(candidate) == product {
			return true
		}
	}
	return false
}

func cloneProviderSpec(spec ProviderSpec) ProviderSpec {
	spec.Products = append([]string(nil), spec.Products...)
	return spec
}
