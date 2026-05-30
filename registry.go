package endpoint

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultRegistryTTL = 15 * time.Minute

// ChangeSummary describes endpoint registry changes from one owner refresh.
type ChangeSummary struct {
	Added   []Ref `json:"added,omitempty"`
	Updated []Ref `json:"updated,omitempty"`
	Removed []Ref `json:"removed,omitempty"`
}

// Registry stores endpoint records in memory.
type Registry struct {
	mu      sync.RWMutex
	ttl     time.Duration
	records map[string]RuntimeRecord
}

// NewRegistry returns an in-memory registry with ttl for new discovered records.
func NewRegistry(ttl time.Duration) *Registry {
	if ttl <= 0 {
		ttl = defaultRegistryTTL
	}
	return &Registry{ttl: ttl, records: map[string]RuntimeRecord{}}
}

// Put stores record and returns its endpoint ref.
func (r *Registry) Put(record RuntimeRecord) (Ref, error) {
	return r.putWithChange(record, nil)
}

// ReplaceOwned stores records for owner and optionally removes previously
// owned records that are absent from records.
func (r *Registry) ReplaceOwned(owner string, records []RuntimeRecord, removeMissing bool) (ChangeSummary, error) {
	if r == nil {
		return ChangeSummary{}, fmt.Errorf("endpoint: registry is nil")
	}
	owner = strings.TrimSpace(owner)
	var summary ChangeSummary
	seen := map[string]bool{}
	for _, record := range records {
		if owner != "" {
			record.Owner = owner
			if record.Metadata == nil {
				record.Metadata = map[string]string{}
			}
			record.Metadata["provider"] = owner
		}
		ref, err := r.putWithChange(record, &summary)
		if err != nil {
			return summary, err
		}
		seen[ref.ID()] = true
	}
	if removeMissing && owner != "" {
		r.mu.Lock()
		for id, record := range r.records {
			if record.Owner == owner && !seen[id] {
				delete(r.records, id)
				summary.Removed = append(summary.Removed, NewRef(id))
			}
		}
		r.mu.Unlock()
	}
	sortRefs(summary.Added)
	sortRefs(summary.Updated)
	sortRefs(summary.Removed)
	return summary, nil
}

func (r *Registry) putWithChange(record RuntimeRecord, summary *ChangeSummary) (Ref, error) {
	if r == nil {
		return "", fmt.Errorf("endpoint: registry is nil")
	}
	name := strings.TrimSpace(record.Spec.Name)
	if name == "" {
		name = Ref(record.Resolved.Ref).ID()
	}
	if name == "" {
		return "", fmt.Errorf("endpoint: record name is empty")
	}
	if record.Discovered.IsZero() {
		record.Discovered = time.Now()
	}
	if record.Expires.IsZero() {
		record.Expires = record.Discovered.Add(r.ttl)
	}
	ref := NewRef(name)
	record.Resolved.Ref = ref
	if strings.TrimSpace(record.Resolved.URL) == "" {
		record.Resolved.URL = record.Spec.URL
	}
	if strings.TrimSpace(record.Resolved.AuthRef) == "" {
		record.Resolved.AuthRef = record.Spec.AuthRef
	}
	if record.Resolved.Source.Kind == "" {
		record.Resolved.Source = record.Source
	}
	if len(record.Resolved.Metadata) == 0 {
		record.Resolved.Metadata = cloneMap(record.Metadata)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := ref.ID()
	previous, existed := r.records[id]
	r.records[id] = cloneRuntimeRecord(record)
	if summary != nil {
		switch {
		case !existed:
			summary.Added = append(summary.Added, ref)
		case !recordsEquivalent(previous, record):
			summary.Updated = append(summary.Updated, ref)
		}
	}
	return ref, nil
}

// Resolve returns a fresh resolved endpoint.
func (r *Registry) Resolve(ref Ref) (Resolved, bool) {
	if r == nil || !ref.Valid() {
		return Resolved{}, false
	}
	r.mu.RLock()
	record, ok := r.records[ref.ID()]
	r.mu.RUnlock()
	if !ok || (!record.Expires.IsZero() && time.Now().After(record.Expires)) {
		return Resolved{}, false
	}
	resolved := record.Resolved
	if strings.TrimSpace(resolved.ExpiresAt) == "" && !record.Expires.IsZero() {
		resolved.ExpiresAt = record.Expires.UTC().Format(time.RFC3339)
	}
	return resolved, true
}

// List returns fresh endpoint records matching product.
func (r *Registry) List(product string) []RuntimeRecord {
	if r == nil {
		return nil
	}
	product = strings.TrimSpace(product)
	now := time.Now()
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []RuntimeRecord
	for _, record := range r.records {
		if !record.Expires.IsZero() && now.After(record.Expires) {
			continue
		}
		if product != "" && record.Spec.Product != product && record.Resolved.Metadata["product"] != product {
			continue
		}
		out = append(out, cloneRuntimeRecord(record))
	}
	return out
}

func cloneRuntimeRecord(record RuntimeRecord) RuntimeRecord {
	record.Spec.Labels = cloneMap(record.Spec.Labels)
	record.Spec.Annotations = cloneMap(record.Spec.Annotations)
	record.Resolved.Headers = cloneMap(record.Resolved.Headers)
	record.Resolved.Metadata = cloneMap(record.Resolved.Metadata)
	record.Resolved.Source.Attributes = cloneMap(record.Resolved.Source.Attributes)
	record.Source.Attributes = cloneMap(record.Source.Attributes)
	record.Metadata = cloneMap(record.Metadata)
	return record
}

func recordsEquivalent(a, b RuntimeRecord) bool {
	a.Discovered = time.Time{}
	b.Discovered = time.Time{}
	a.Expires = time.Time{}
	b.Expires = time.Time{}
	return reflect.DeepEqual(a, b)
}

func sortRefs(refs []Ref) {
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
}
