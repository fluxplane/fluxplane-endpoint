package endpoint

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Ref identifies an endpoint stored or resolved by runtime code. The canonical
// string form is @endpoint/<id>.
type Ref string

// EndpointRefPrefix is the canonical prefix for endpoint refs.
const EndpointRefPrefix = "@endpoint/"

// NewRef returns the canonical endpoint ref for id.
func NewRef(id string) Ref {
	id = strings.TrimSpace(strings.TrimPrefix(id, EndpointRefPrefix))
	if id == "" {
		return ""
	}
	return Ref(EndpointRefPrefix + id)
}

// ParseRef parses a canonical endpoint ref or bare id.
func ParseRef(value string) Ref {
	return NewRef(value)
}

// ID returns the ref id without the @endpoint/ prefix.
func (r Ref) ID() string {
	return strings.TrimSpace(strings.TrimPrefix(string(r), EndpointRefPrefix))
}

// Valid reports whether r is a non-empty endpoint ref.
func (r Ref) Valid() bool {
	return strings.HasPrefix(strings.TrimSpace(string(r)), EndpointRefPrefix) && r.ID() != ""
}

// SourceRef describes where an endpoint came from without importing the source
// system into endpoint contracts.
type SourceRef struct {
	Kind       string            `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name       string            `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Cluster    string            `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	UID        string            `json:"uid,omitempty" yaml:"uid,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}

// Normalize returns a source ref with trimmed scalar fields.
func (s SourceRef) Normalize() SourceRef {
	s.Kind = strings.TrimSpace(s.Kind)
	s.Name = strings.TrimSpace(s.Name)
	s.Namespace = strings.TrimSpace(s.Namespace)
	s.Cluster = strings.TrimSpace(s.Cluster)
	s.UID = strings.TrimSpace(s.UID)
	s.Attributes = cloneMap(s.Attributes)
	return s
}

// EndpointSpec declares a manifest-level endpoint capability. It describes the
// kind of endpoints a provider can use or discover, not one configured URL.
type EndpointSpec struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Products    []string `json:"products,omitempty" yaml:"products,omitempty"`
	Env         []string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Spec describes an explicitly configured endpoint.
type Spec struct {
	Name        string            `json:"name" yaml:"name"`
	URL         string            `json:"url,omitempty" yaml:"url,omitempty"`
	Product     string            `json:"product,omitempty" yaml:"product,omitempty"`
	Protocol    string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	AuthRef     string            `json:"auth_ref,omitempty" yaml:"auth_ref,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// Normalize returns a spec with trimmed scalar fields and cloned maps.
func (s Spec) Normalize() Spec {
	s.Name = strings.TrimSpace(s.Name)
	s.URL = strings.TrimRight(strings.TrimSpace(s.URL), "/")
	s.Product = strings.TrimSpace(s.Product)
	s.Protocol = strings.TrimSpace(s.Protocol)
	s.AuthRef = strings.TrimSpace(s.AuthRef)
	s.Labels = cloneMap(s.Labels)
	s.Annotations = cloneMap(s.Annotations)
	return s
}

// Validate checks the configured endpoint has an identity and target.
func (s Spec) Validate() error {
	s = s.Normalize()
	if s.Name == "" {
		return fmt.Errorf("endpoint: name is empty")
	}
	if s.URL == "" {
		return fmt.Errorf("endpoint: url is empty")
	}
	return ValidateURL(s.URL)
}

// EndpointRef is the wire/storage endpoint representation used by host/plugin
// protocols. It intentionally carries credential references, not secret values.
type EndpointRef struct {
	ID            string            `json:"id" yaml:"id"`
	URL           string            `json:"url" yaml:"url"`
	Product       string            `json:"product,omitempty" yaml:"product,omitempty"`
	Protocol      string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Source        string            `json:"source,omitempty" yaml:"source,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// Normalize returns a wire endpoint ref with trimmed scalar fields, inferred
// protocol, and a generated id when possible.
func (r EndpointRef) Normalize() EndpointRef {
	r.ID = strings.TrimSpace(r.ID)
	r.URL = strings.TrimRight(strings.TrimSpace(r.URL), "/")
	r.Product = strings.TrimSpace(r.Product)
	r.Protocol = strings.TrimSpace(r.Protocol)
	r.Source = strings.TrimSpace(r.Source)
	r.CredentialRef = strings.TrimSpace(r.CredentialRef)
	r.Labels = cloneMap(r.Labels)
	r.Annotations = cloneMap(r.Annotations)
	if r.Protocol == "" {
		if parsed, err := url.Parse(r.URL); err == nil {
			r.Protocol = parsed.Scheme
		}
	}
	if r.Source == "" {
		r.Source = "manual"
	}
	if r.ID == "" {
		r.ID = IDFromProductURL(r.Product, r.URL)
	}
	return r
}

// Validate checks that r has an id and a syntactically usable endpoint URL.
func (r EndpointRef) Validate() error {
	r = r.Normalize()
	if r.ID == "" {
		return fmt.Errorf("endpoint: id is empty")
	}
	if r.URL == "" {
		return fmt.Errorf("endpoint: url is empty")
	}
	return ValidateURL(r.URL)
}

// CanonicalRef returns the @endpoint/<id> ref for r.
func (r EndpointRef) CanonicalRef() Ref {
	return NewRef(r.Normalize().ID)
}

// Resolved is the runtime-ready view of an endpoint. Headers may identify
// trusted runtime material; callers must keep them out of model-visible logs.
type Resolved struct {
	Ref        Ref               `json:"ref,omitempty" yaml:"ref,omitempty"`
	URL        string            `json:"url" yaml:"url"`
	HeadersRef string            `json:"headers_ref,omitempty" yaml:"headers_ref,omitempty"`
	AuthRef    string            `json:"auth_ref,omitempty" yaml:"auth_ref,omitempty"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	ExpiresAt  string            `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Source     SourceRef         `json:"source,omitempty" yaml:"source,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Normalize returns a resolved endpoint with trimmed scalar fields and cloned maps.
func (r Resolved) Normalize() Resolved {
	r.Ref = NewRef(r.Ref.ID())
	r.URL = strings.TrimRight(strings.TrimSpace(r.URL), "/")
	r.HeadersRef = strings.TrimSpace(r.HeadersRef)
	r.AuthRef = strings.TrimSpace(r.AuthRef)
	r.Headers = cloneMap(r.Headers)
	r.Source = r.Source.Normalize()
	r.Metadata = cloneMap(r.Metadata)
	return r
}

// Candidate is the host/plugin protocol endpoint discovery candidate. It keeps
// Source as a string for stable JSON compatibility with dex plugins.
type Candidate struct {
	ID            string            `json:"id" yaml:"id"`
	URL           string            `json:"url,omitempty" yaml:"url,omitempty"`
	Product       string            `json:"product,omitempty" yaml:"product,omitempty"`
	Protocol      string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Source        string            `json:"source,omitempty" yaml:"source,omitempty"`
	Score         float64           `json:"score,omitempty" yaml:"score,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// Normalize returns a candidate with trimmed scalar fields and cloned maps.
func (c Candidate) Normalize() Candidate {
	c.ID = strings.TrimSpace(c.ID)
	c.URL = strings.TrimRight(strings.TrimSpace(c.URL), "/")
	c.Product = strings.TrimSpace(c.Product)
	c.Protocol = strings.TrimSpace(c.Protocol)
	c.Source = strings.TrimSpace(c.Source)
	c.CredentialRef = strings.TrimSpace(c.CredentialRef)
	c.Labels = cloneMap(c.Labels)
	c.Annotations = cloneMap(c.Annotations)
	if c.Protocol == "" {
		if parsed, err := url.Parse(c.URL); err == nil {
			c.Protocol = parsed.Scheme
		}
	}
	if c.ID == "" {
		c.ID = IDFromProductURL(c.Product, c.URL)
	}
	return c
}

// EndpointRef converts c to a stored endpoint ref.
func (c Candidate) EndpointRef() EndpointRef {
	c = c.Normalize()
	return EndpointRef{ID: c.ID, URL: c.URL, Product: c.Product, Protocol: c.Protocol, Source: c.Source, CredentialRef: c.CredentialRef, Labels: cloneMap(c.Labels), Annotations: cloneMap(c.Annotations)}.Normalize()
}

// DiscoveryCandidate is the richer core discovery candidate with structured
// source metadata and matching/probe hints.
type DiscoveryCandidate struct {
	ID          string            `json:"id" yaml:"id"`
	URL         string            `json:"url,omitempty" yaml:"url,omitempty"`
	Scheme      string            `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Host        string            `json:"host,omitempty" yaml:"host,omitempty"`
	Port        int               `json:"port,omitempty" yaml:"port,omitempty"`
	PortName    string            `json:"port_name,omitempty" yaml:"port_name,omitempty"`
	ProductHint string            `json:"product_hint,omitempty" yaml:"product_hint,omitempty"`
	Protocol    string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	AuthRef     string            `json:"auth_ref,omitempty" yaml:"auth_ref,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Source      SourceRef         `json:"source" yaml:"source"`
	Reasons     []string          `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	Score       float64           `json:"score,omitempty" yaml:"score,omitempty"`
}

// Normalize returns a discovery candidate with trimmed scalar fields and cloned maps.
func (c DiscoveryCandidate) Normalize() DiscoveryCandidate {
	c.ID = strings.TrimSpace(c.ID)
	c.URL = strings.TrimRight(strings.TrimSpace(c.URL), "/")
	c.Scheme = strings.TrimSpace(c.Scheme)
	c.Host = strings.TrimSpace(c.Host)
	c.PortName = strings.TrimSpace(c.PortName)
	c.ProductHint = strings.TrimSpace(c.ProductHint)
	c.Protocol = strings.TrimSpace(c.Protocol)
	c.AuthRef = strings.TrimSpace(c.AuthRef)
	c.Labels = cloneMap(c.Labels)
	c.Annotations = cloneMap(c.Annotations)
	c.Source = c.Source.Normalize()
	c.Reasons = trimStrings(c.Reasons)
	if c.Protocol == "" {
		c.Protocol = firstNonEmpty(c.Scheme, schemeOf(c.URL))
	}
	if c.ID == "" {
		c.ID = IDFromProductURL(c.ProductHint, c.URL)
	}
	return c
}

// Candidate converts c to the host/plugin protocol candidate shape.
func (c DiscoveryCandidate) Candidate() Candidate {
	c = c.Normalize()
	return Candidate{ID: c.ID, URL: c.URL, Product: c.ProductHint, Protocol: c.Protocol, Source: c.Source.Kind, Score: c.Score, CredentialRef: c.AuthRef, Labels: cloneMap(c.Labels), Annotations: cloneMap(c.Annotations)}.Normalize()
}

// EndpointRef converts c to a stored endpoint ref.
func (c DiscoveryCandidate) EndpointRef() EndpointRef {
	return c.Candidate().EndpointRef()
}

// DetectorSpec declares product-neutral candidate matching hints.
type DetectorSpec struct {
	Product      string              `json:"product" yaml:"product"`
	Names        []string            `json:"names,omitempty" yaml:"names,omitempty"`
	Labels       map[string][]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Ports        []int               `json:"ports,omitempty" yaml:"ports,omitempty"`
	PortNames    []string            `json:"port_names,omitempty" yaml:"port_names,omitempty"`
	Schemes      []string            `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Protocols    []string            `json:"protocols,omitempty" yaml:"protocols,omitempty"`
	Sources      []string            `json:"sources,omitempty" yaml:"sources,omitempty"`
	ExcludeNames []string            `json:"exclude_names,omitempty" yaml:"exclude_names,omitempty"`
}

// ProbeSpec declares a safe probe. This package does not execute probes.
type ProbeSpec struct {
	Product       string            `json:"product" yaml:"product"`
	Method        string            `json:"method,omitempty" yaml:"method,omitempty"`
	Path          string            `json:"path" yaml:"path"`
	ExpectedCodes []int             `json:"expected_codes,omitempty" yaml:"expected_codes,omitempty"`
	Timeout       string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers       map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ProbeResult records the outcome of an executed probe.
type ProbeResult struct {
	CandidateID string            `json:"candidate_id" yaml:"candidate_id"`
	Probe       ProbeSpec         `json:"probe" yaml:"probe"`
	Status      string            `json:"status" yaml:"status"`
	LatencyMS   int64             `json:"latency_ms,omitempty" yaml:"latency_ms,omitempty"`
	Product     string            `json:"product,omitempty" yaml:"product,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Error       string            `json:"error,omitempty" yaml:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// DiscoveryRequest describes one endpoint discovery request.
type DiscoveryRequest struct {
	Op        string            `json:"op,omitempty" yaml:"op,omitempty"`
	Providers []string          `json:"providers,omitempty" yaml:"providers,omitempty"`
	Product   string            `json:"product,omitempty" yaml:"product,omitempty"`
	Products  []string          `json:"products,omitempty" yaml:"products,omitempty"`
	Query     map[string]string `json:"query,omitempty" yaml:"query,omitempty"`
	Limit     int               `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// DiscoveryResult is one endpoint discovery response.
type DiscoveryResult struct {
	EndpointRefs []Ref                `json:"endpoint_refs,omitempty" yaml:"endpoint_refs,omitempty"`
	Candidates   []DiscoveryCandidate `json:"candidates,omitempty" yaml:"candidates,omitempty"`
	Probes       []ProbeResult        `json:"probes,omitempty" yaml:"probes,omitempty"`
}

// ProviderSpec describes one registered discovery provider.
type ProviderSpec struct {
	Name        string   `json:"name" yaml:"name"`
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
	Products    []string `json:"products,omitempty" yaml:"products,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// ProviderStatus is the current status for one discovery provider.
type ProviderStatus struct {
	Spec        ProviderSpec `json:"spec" yaml:"spec"`
	LastRun     string       `json:"last_run,omitempty" yaml:"last_run,omitempty"`
	LastError   string       `json:"last_error,omitempty" yaml:"last_error,omitempty"`
	LastResults int          `json:"last_results,omitempty" yaml:"last_results,omitempty"`
}

// Health captures a non-secret endpoint probe result.
type Health struct {
	OK         bool              `json:"ok" yaml:"ok"`
	CheckedAt  time.Time         `json:"checked_at" yaml:"checked_at"`
	Method     string            `json:"method,omitempty" yaml:"method,omitempty"`
	DurationMS int64             `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
	Error      string            `json:"error,omitempty" yaml:"error,omitempty"`
	Details    map[string]any    `json:"details,omitempty" yaml:"details,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Record is a stored endpoint record with health and lifecycle metadata.
type Record struct {
	EndpointRef `json:",inline" yaml:",inline"`
	LastHealth  *Health   `json:"last_health,omitempty" yaml:"last_health,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

// RuntimeRecord is one in-memory runtime endpoint registry entry.
type RuntimeRecord struct {
	Spec       Spec              `json:"spec,omitempty" yaml:"spec,omitempty"`
	Resolved   Resolved          `json:"resolved,omitempty" yaml:"resolved,omitempty"`
	Source     SourceRef         `json:"source,omitempty" yaml:"source,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Owner      string            `json:"owner,omitempty" yaml:"owner,omitempty"`
	Discovered time.Time         `json:"discovered,omitempty" yaml:"discovered,omitempty"`
	Expires    time.Time         `json:"expires,omitempty" yaml:"expires,omitempty"`
}

// ValidateURL checks endpoint URL syntax while allowing file-like schemes used
// by SQLite endpoints.
func ValidateURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" && !URLAllowsEmptyHost(parsed.Scheme) {
		return fmt.Errorf("endpoint: invalid url %q", raw)
	}
	return nil
}

// URLAllowsEmptyHost reports whether scheme endpoints may omit URL host.
func URLAllowsEmptyHost(scheme string) bool {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "file", "sqlite", "sqlite3":
		return true
	default:
		return false
	}
}

// IDFromProductURL returns a stable endpoint id from product and rawURL.
func IDFromProductURL(product, rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	host := ""
	if err == nil {
		host = parsed.Host
	}
	id := strings.Trim(strings.ToLower(strings.TrimSpace(product)+"-"+host), "-")
	if id == "" {
		id = strings.TrimSpace(rawURL)
	}
	replacer := strings.NewReplacer("://", "-", "/", "-", ":", "-", ".", "-", "_", "-")
	id = replacer.Replace(id)
	id = strings.Trim(id, "-")
	if id == "" {
		id = "endpoint"
	}
	return id
}

func schemeOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return parsed.Scheme
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func trimStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
