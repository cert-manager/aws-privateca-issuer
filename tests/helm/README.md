# Helm Chart End-to-End Tests

This directory contains end-to-end tests for the AWS Private CA Issuer Helm chart that deploy to a real Kind cluster and validate actual Kubernetes resources.

## Test Strategy

The testing approach follows existing e2e patterns in the repository:
- **End-to-End Deployment Testing**: Charts are actually deployed to Kind cluster
- **Real Resource Validation**: Tests verify actual Kubernetes resources are created correctly
- **Conditional Logic Testing**: All branching logic tested through deployment scenarios
- **Value Substitution Validation**: Real deployments ensure `{{ .Values.* }}` work correctly

## Dependencies

### Why Separate go.mod?

This test directory has its own `go.mod` and `go.sum` files because:

1. **Test-Specific Dependencies**: Tests require heavy dependencies (Helm SDK, testify) not needed by main application
2. **Version Isolation**: Tests can use different versions of shared dependencies without affecting main app
3. **Build Separation**: Tests build independently without polluting main binary with test-only dependencies

Key test dependencies:
```go
require (
    github.com/stretchr/testify v1.8.4    // Test assertions
    helm.sh/helm/v3 v3.16.4               // Helm Go SDK for deployments
    k8s.io/client-go v0.31.2              // Kubernetes client for validation
)
```

## Running Tests

### Primary Method (Recommended)

```bash
# From repository root - sets up Kind cluster, cert-manager, and runs tests
make e2eHelmTest
```

### Direct Method

```bash
# From tests/helm directory - requires existing cluster setup
./run-tests.sh
```

## Test Structure

```
tests/helm/
├── autoscaling_test.go      # Tests HPA and replica count logic
├── rbac_test.go            # Tests RBAC resource creation
├── deployment_test.go      # Tests deployment configuration
├── service_monitor_test.go # Tests ServiceMonitor and PDB
├── common_test.go          # Shared test utilities
├── go.mod                  # Dependencies
├── run-tests.sh           # Direct test runner
└── README.md              # This file
```

## What Gets Tested

All conditional logic in Helm templates through actual deployments:

### Autoscaling (`autoscaling_test.go`)
- HPA creation when `autoscaling.enabled=true`
- Replica count removal from Deployment when autoscaling enabled
- CPU and memory target configuration

### RBAC (`rbac_test.go`)
- ClusterRole and ClusterRoleBinding creation
- ServiceAccount creation with annotations
- Approver role configuration for cert-manager

### Deployment (`deployment_test.go`)
- Command line flags (`disableApprovedCheck`, `disableClientSideRateLimiting`)
- Priority class configuration
- Environment variable injection
- Volume and volume mount configuration
- Sidecar container addition

### Service Monitor (`service_monitor_test.go`)
- ServiceMonitor creation for Prometheus
- PodDisruptionBudget configuration

## How It Works

1. **Makefile Integration**: `e2eHelmTest` target uses existing `kind-cluster` and `deploy-cert-manager` dependencies
2. **Real Deployment**: Tests use Helm Go SDK to actually install charts to Kind cluster
3. **Resource Validation**: Tests use Kubernetes client-go to verify resources exist and are configured correctly
4. **Cleanup**: Each test cleans up its resources after validation

## Adding New Tests

When adding new conditional logic to templates:

1. **Add test case** in appropriate test file
2. **Deploy with values** that trigger the conditional logic
3. **Validate resources** using Kubernetes client-go
4. **Test both enabled/disabled** states

Example:
```go
{
    name: "newFeature enabled creates expected resource",
    values: map[string]interface{}{
        "newFeature": map[string]interface{}{
            "enabled": true,
        },
    },
    validate: func(t *testing.T, h *testHelper) {
        // Verify actual Kubernetes resource exists
        resource, err := h.clientset.AppsV1().Deployments(h.namespace).Get(...)
        require.NoError(t, err)
        assert.Equal(t, expectedValue, resource.Spec.SomeField)
    },
},
```

## Benefits

- **Regression Prevention**: Real deployments catch template and configuration issues
- **Comprehensive Validation**: Tests both template rendering and runtime behavior
- **CI/CD Integration**: Automated testing on every chart change
- **Familiar Patterns**: Uses same approach as existing e2e tests in repository
