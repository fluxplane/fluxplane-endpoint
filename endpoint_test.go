package endpoint

import "testing"

func TestRefCanonicalization(t *testing.T) {
	ref := NewRef(" dev ")
	if ref != "@endpoint/dev" || ref.ID() != "dev" || !ref.Valid() {
		t.Fatalf("ref = %q id=%q valid=%v", ref, ref.ID(), ref.Valid())
	}
	if ParseRef("@endpoint/dev") != ref {
		t.Fatalf("ParseRef did not round-trip")
	}
}

func TestEndpointRefNormalizeAndValidate(t *testing.T) {
	ref := EndpointRef{URL: " https://gitlab.example.com/ ", Product: " gitlab "}.Normalize()
	if ref.ID != "gitlab-gitlab-example-com" || ref.Protocol != "https" || ref.Source != "manual" {
		t.Fatalf("normalized ref = %#v", ref)
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if ref.CanonicalRef() != "@endpoint/gitlab-gitlab-example-com" {
		t.Fatalf("canonical ref = %q", ref.CanonicalRef())
	}
}

func TestEndpointSpecCarriesURLAliases(t *testing.T) {
	spec := EndpointSpec{Name: "gitlab.endpoint", Products: []string{"gitlab"}, Env: []string{"GITLAB_URL"}}
	if len(spec.Env) != 1 || spec.Env[0] != "GITLAB_URL" {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestDiscoveryCandidateConversion(t *testing.T) {
	candidate := DiscoveryCandidate{
		URL:         "http://loki.monitoring.svc:3100",
		ProductHint: "loki",
		Source:      SourceRef{Kind: "kubernetes_service", Namespace: "monitoring"},
		AuthRef:     "kubernetes://monitoring/secrets/loki",
		Reasons:     []string{" service ", ""},
	}.Normalize()
	if candidate.ID == "" || candidate.Protocol != "http" || len(candidate.Reasons) != 1 {
		t.Fatalf("normalized candidate = %#v", candidate)
	}
	wire := candidate.Candidate()
	if wire.Product != "loki" || wire.Source != "kubernetes_service" || wire.CredentialRef != candidate.AuthRef {
		t.Fatalf("wire candidate = %#v", wire)
	}
}

func TestDiscoveryCandidateNormalizeDoesNotMutateReasons(t *testing.T) {
	reasons := []string{" service ", ""}
	candidate := DiscoveryCandidate{URL: "http://example.com", ProductHint: "demo", Reasons: reasons}.Normalize()
	if len(candidate.Reasons) != 1 || candidate.Reasons[0] != "service" {
		t.Fatalf("normalized reasons = %#v", candidate.Reasons)
	}
	if reasons[0] != " service " || reasons[1] != "" {
		t.Fatalf("input reasons mutated: %#v", reasons)
	}
}

func TestValidateURLAllowsSQLite(t *testing.T) {
	if err := ValidateURL("sqlite:///tmp/dev.db"); err != nil {
		t.Fatalf("sqlite URL should be valid: %v", err)
	}
	if err := ValidateURL("http:///missing-host"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
