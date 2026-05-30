# fluxplane-endpoint

Shared endpoint discovery and registry contracts for Fluxplane modules.

`fluxplane-endpoint` defines portable endpoint data structures and small in-memory helpers that can be shared by Fluxplane runtimes, plugins, and external tooling without importing `fluxplane-core`. It covers configured endpoints, discovered endpoint candidates, provider status, runtime resolution, and refresh summaries while keeping credentials and host-specific probing outside this module.

## Install

```sh
go get github.com/fluxplane/fluxplane-endpoint
```

## Usage

### Configure or store an endpoint

```go
import endpoint "github.com/fluxplane/fluxplane-endpoint"

spec := endpoint.Spec{
    Name:    "grafana-local",
    URL:     "http://localhost:3000",
    Product: "grafana",
}
if err := spec.Validate(); err != nil {
    return err
}

registry := endpoint.NewRegistry(0)
ref, err := registry.Put(endpoint.RuntimeRecord{Spec: spec})
if err != nil {
    return err
}
resolved, ok := registry.Resolve(ref)
_ = resolved
_ = ok
```

### Register discovery providers

```go
import endpoint "github.com/fluxplane/fluxplane-endpoint"

providers := endpoint.NewDiscoveryRegistry()
// Register implementations of endpoint.DiscoveryProvider from host/runtime code.
// providers.Register(kubernetesProvider)

runner := endpoint.NewRunner(providers, endpoint.NewRegistry(0))
summary := runner.DiscoverNow(ctx, endpoint.RunRequest{
    Products: []string{"grafana"},
    Reason:   "manual refresh",
})
_ = summary
```

## Key types

- `Ref` and `EndpointRef` define canonical `@endpoint/<id>` references and serializable endpoint records.
- `Spec` describes explicitly configured endpoints.
- `Resolved` is the runtime-ready view with non-secret headers and auth references.
- `Candidate` and `DiscoveryCandidate` represent endpoint discovery output from host/plugin providers.
- `DetectorSpec`, `ProbeSpec`, and `ProbeResult` describe discovery/probe metadata; this package does not execute probes.
- `DiscoveryProvider` and `DiscoveryRegistry` coordinate provider registration and discovery status.
- `Registry` stores fresh runtime endpoint records in memory and resolves references.
- `Runner`, `RunRequest`, and `RunSummary` refresh discovery providers into a registry and report added, updated, and removed endpoints.

## Design

This module intentionally stays policy-neutral and dependency-light. It should not know about Fluxplane workspaces, plugin hosts, credential storage, Kubernetes clients, HTTP probing, or authorization policy. Higher-level modules provide concrete discovery providers, execute probes, resolve credentials, and enforce safety boundaries.

The module is versioned independently so downstream projects can share endpoint contracts without depending on Fluxplane runtime implementations.
