# Upstream patch: recognize int32/int64 in CRD format validation

## Problem

When running e2e (or any `clusterctl init` against a cluster with CRDs that use OpenAPI `format: int32` or `format: int64`), the Kubernetes API server logs INFO lines like:

```
INFO	unrecognized format "int32"
INFO	unrecognized format "int64"
```

These come from **k8s.io/apiextensions-apiserver** in the API server process (e.g. inside the Kind node). The code path is:

1. CRDs are applied to the cluster.
2. `pkg/registry/customresourcedefinition/strategy.go` calls `getUnrecognizedFormatsInSchema` for each schema.
3. That uses `apiservervalidation.GetUnrecognizedFormats()` in `pkg/apiserver/validation/formats.go`.
4. The supported-format list in `formats.go` only includes string-related formats (uri, email, uuid, etc.), so `int32` and `int64` are reported as unrecognized and surfaced as warnings.

Behaviour is correct (validation still accepts the values); only the warning is noisy.

## Fix (upstream Kubernetes)

**Repository:** [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)  
**File:** `staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/validation/formats.go`

In the first `versionedFormats` entry (the one with `introducedVersion: version.MajorMinor(1, 0)`), add the standard OpenAPI integer formats to the `formats` set so they are not reported as unrecognized:

```go
// In supportedVersionedFormats, first entry's formats set, add:
			"int32",         // standard OpenAPI format for 32-bit integer
			"int64",         // standard OpenAPI format for 64-bit integer
```

So the first `formats: sets.New(...)` block gains two more strings: `"int32"` and `"int64"`.

## Applying the change

1. Clone kubernetes/kubernetes and open `staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/validation/formats.go`.
2. In the first `versionedFormats` struct (around line 32), inside the first `sets.New(...)` call, add:
   - `"int32",`
   - `"int64",`
3. Run the package tests: `go test ./staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/validation/...`
4. Open a PR against kubernetes/kubernetes with the change and reference that CRD schema validation uses OpenAPI formats and that int32/int64 are standard and should not be warned on.

## Why we don’t patch it in this repo

The process that prints the warning is the **API server binary** running inside the Kind node image (e.g. `kindest/node:v1.34.0`). That binary is built from kubernetes/kubernetes, not from cluster-api-provider-microvm. Changing our go.mod or vendoring apiextensions-apiserver here does not change the Kind node’s API server, so the only way to remove the warning is to fix it upstream and use a Kubernetes/Kind release that includes the fix.
