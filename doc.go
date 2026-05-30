// Package endpoint defines shared, inert endpoint contracts for Fluxplane hosts,
// plugins, discovery providers, and registries.
//
// Endpoint contracts describe service locations, discovery candidates, source
// metadata, and non-secret health/status records. The package does not dial
// networks, probe services, resolve credentials, or persist registry state.
package endpoint
