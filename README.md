# fluxplane-endpoint

Shared endpoint discovery contract types for Fluxplane modules.

This module contains portable endpoint descriptors and helpers used to exchange discovered service information between Fluxplane runtimes, plugins, and external tooling without importing `fluxplane-core`.

## Usage

```go
import endpoint "github.com/fluxplane/fluxplane-endpoint"

ep := endpoint.Endpoint{
    ID:   "service:api",
    Kind: "http",
    URL:  "https://api.example.com",
}
```

## Packages

- `endpoint.Endpoint` describes a discovered service endpoint.
- Endpoint metadata is intentionally plain Go data so it can be serialized, indexed, and shared across module boundaries.

This package is versioned independently for consumers that only need endpoint discovery contracts.
