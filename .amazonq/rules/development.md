# AWS Private CA Issuer - Project Structure Guide

## Overview
This is a Kubernetes operator that integrates AWS Private Certificate Authority (PCA) with cert-manager, allowing automatic certificate issuance from AWS PCA. Built using Kubebuilder v3, it's a standard Kubernetes controller pattern implementation.

**Repository**: github.com/cert-manager/aws-privateca-issuer  
**Language**: Go 1.23+  
**Framework**: Kubebuilder v3 / controller-runtime  

---

## Project Architecture

### Core Components

1. **Custom Resource Definitions (CRDs)**
   - `AWSPCAIssuer` - Namespace-scoped issuer
   - `AWSPCAClusterIssuer` - Cluster-scoped issuer
   - Both share the same spec (`AWSPCAIssuerSpec`) and status (`AWSPCAIssuerStatus`)

2. **Controllers**
   - `AWSPCAIssuerReconciler` - Manages AWSPCAIssuer resources
   - `AWSPCAClusterIssuerReconciler` - Manages AWSPCAClusterIssuer resources
   - `GenericIssuerReconciler` - Shared logic for both issuer types
   - `CertificateRequestReconciler` - Handles cert-manager CertificateRequest resources

3. **AWS Integration**
   - AWS SDK v2 for Go
   - ACM PCA client for certificate operations
   - STS for role assumption (cross-account support)

---

## Directory Structure

```
aws-privateca-issuer/
├── main.go                    # Entry point - sets up controllers and manager
├── go.mod/go.sum             # Go module dependencies
├── Makefile                  # Build, test, and deployment automation
├── Dockerfile                # Container image build
├── PROJECT                   # Kubebuilder project metadata
│
├── pkg/                      # Core application code
│   ├── api/                  # API definitions
│   │   ├── v1beta1/         # CRD types and interfaces
│   │   │   ├── awspcaissuer_types.go      # CRD struct definitions
│   │   │   ├── generic_issuer.go          # GenericIssuer interface
│   │   │   └── zz_generated.deepcopy.go   # Auto-generated deepcopy methods
│   │   ├── config/          # Configuration injection targets
│   │   └── injections/      # Version/UserAgent injection
│   ├── controllers/         # Reconciliation logic
│   │   ├── awspcaissuer_controller.go           # AWSPCAIssuer controller
│   │   ├── awspcaclusterissuer_controller.go    # AWSPCAClusterIssuer controller
│   │   ├── genericissuer_controller.go          # Shared issuer logic
│   │   ├── certificaterequest_controller.go     # Certificate signing logic
│   │   └── *_test.go                            # Unit tests
│   ├── aws/                 # AWS PCA integration
│   │   ├── pca.go          # PCA client, provisioner, certificate signing
│   │   └── pca_test.go     # AWS integration tests
│   ├── util/               # Utility functions
│   │   └── issuers.go      # Issuer helper functions
│   └── clientset/          # Kubernetes client wrappers
│       └── v1beta1/        # Typed clients for CRDs
│
├── config/                  # Kustomize-based Kubernetes manifests
│   ├── default/            # Main kustomization (combines all below)
│   │   └── kustomization.yaml
│   ├── crd/                # Custom Resource Definitions
│   │   ├── bases/          # Generated CRD YAML files
│   │   │   ├── awspca.cert-manager.io_awspcaissuers.yaml
│   │   │   └── awspca.cert-manager.io_awspcaclusterissuers.yaml
│   │   ├── patches/        # CRD customization patches
│   │   └── kustomization.yaml
│   ├── rbac/               # Role-Based Access Control
│   │   ├── role.yaml                    # ClusterRole for controller
│   │   ├── role_binding.yaml            # ClusterRoleBinding
│   │   ├── leader_election_role.yaml    # Leader election permissions
│   │   ├── auth_proxy_*.yaml            # Metrics auth proxy
│   │   ├── cert_manager_controller_approver_*.yaml  # Cert-manager integration
│   │   └── kustomization.yaml
│   ├── manager/            # Controller deployment
│   │   ├── manager.yaml                 # Deployment manifest
│   │   ├── controller_manager_config.yaml
│   │   └── kustomization.yaml
│   ├── prometheus/         # Prometheus monitoring
│   │   ├── monitor.yaml    # ServiceMonitor CRD
│   │   └── kustomization.yaml
│   ├── certmanager/        # Cert-manager integration configs
│   ├── samples/            # Example CRs for testing
│   │   ├── awspcaissuer_rsa/           # RSA issuer examples
│   │   ├── awspcaissuer_ec/            # EC issuer examples
│   │   ├── awspcaclusterissuer_rsa/    # RSA cluster issuer examples
│   │   ├── awspcaclusterissuer_ec/     # EC cluster issuer examples
│   │   └── secret.yaml                 # AWS credentials secret example
│   ├── examples/           # Usage examples
│   │   ├── minimal/                    # Minimal issuer setup
│   │   ├── certificates/               # Certificate examples (RSA, ECDSA)
│   │   ├── config/                     # Configuration examples
│   │   ├── cluster-issuer/             # Cluster issuer examples
│   │   └── cluster-issuer-with-assumption-role/  # Cross-account examples
│   └── scorecard/          # Operator SDK scorecard tests
│
├── charts/                  # Helm chart for deployment
│   └── aws-pca-issuer/     # Helm chart package
│       ├── Chart.yaml      # Chart metadata (name, version)
│       ├── values.yaml     # Default configuration values
│       ├── templates/      # Kubernetes manifest templates
│       │   ├── deployment.yaml         # Controller deployment
│       │   ├── service.yaml            # Service definition
│       │   ├── rbac.yaml               # RBAC resources
│       │   ├── hpa.yaml                # Horizontal Pod Autoscaler
│       │   ├── pdb.yaml                # Pod Disruption Budget
│       │   ├── service-monitor.yaml    # Prometheus ServiceMonitor
│       │   └── _helpers.tpl            # Template helper functions
│       ├── crds/           # CRDs (copied from config/crd/bases/)
│       │   ├── awspca.cert-manager.io_awspcaissuers.yaml
│       │   └── awspca.cert-manager.io_awspcaclusterissuers.yaml
│       └── .helmignore     # Files to exclude from chart package
│
├── e2e/                    # End-to-end tests
│   ├── e2e_test.go        # Main e2e test suite
│   ├── helm_test.sh       # Helm installation tests
│   ├── blog_test.sh       # Blog workflow tests
│   └── [utility files]    # Test helpers and utilities
│
├── bin/                    # Compiled binaries (gitignored)
├── testbin/               # Test dependencies (gitignored)
├── hack/                  # Build and development scripts
├── docs/                  # Documentation
├── .github/               # GitHub Actions CI/CD workflows
└── assets/                # Project assets (logo, etc.)
```

---

## Key Files and Their Purpose

### Entry Point
- **`main.go`**: Application entry point
  - Sets up controller-runtime manager
  - Registers all controllers
  - Configures leader election, metrics, health checks
  - Handles command-line flags (`--disable-approved-check`, `--disable-client-side-rate-limiting`)

### API Definitions
- **`pkg/api/v1beta1/awspcaissuer_types.go`**: CRD type definitions
  - `AWSPCAIssuerSpec`: Configuration (ARN, region, credentials, role)
  - `AWSPCAIssuerStatus`: Status conditions
  - `AWSPCAIssuer`: Namespace-scoped resource
  - `AWSPCAClusterIssuer`: Cluster-scoped resource

- **`pkg/api/v1beta1/generic_issuer.go`**: GenericIssuer interface
  - Abstracts over both issuer types
  - Simplifies controller code by treating both types uniformly

### Controllers
- **`pkg/controllers/genericissuer_controller.go`**: Core issuer reconciliation
  - Validates AWS credentials
  - Checks PCA connectivity
  - Updates issuer status (Ready/NotReady)

- **`pkg/controllers/certificaterequest_controller.go`**: Certificate signing
  - Watches cert-manager CertificateRequest resources
  - Validates approval conditions
  - Calls AWS PCA to issue certificates
  - Updates CertificateRequest with signed certificate

- **`pkg/controllers/awspcaissuer_controller.go`**: Namespace-scoped issuer wrapper
- **`pkg/controllers/awspcaclusterissuer_controller.go`**: Cluster-scoped issuer wrapper

### AWS Integration
- **`pkg/aws/pca.go`**: AWS PCA client and provisioner
  - `LoadConfig()`: Loads AWS configuration (credentials, region, role assumption)
  - `PCAProvisioner`: Handles certificate signing
  - `Sign()`: Issues certificate via AWS PCA
  - `Get()`: Retrieves issued certificate
  - Template ARN mapping based on cert-manager usage types

### Deployment Configurations

#### Kustomize (config/)
- **Purpose**: Raw Kubernetes manifests for direct deployment
- **Usage**: `kubectl apply -k config/default`
- **Structure**: Modular bases with patches
- **When to modify**:
  - CRD changes: Regenerate with `make manifests`
  - RBAC changes: Edit `config/rbac/role.yaml`
  - Deployment changes: Edit `config/manager/manager.yaml`

#### Helm (charts/aws-pca-issuer/)
- **Purpose**: Packaged deployment with templating
- **Usage**: `helm install aws-pca-issuer charts/aws-pca-issuer`
- **Structure**: Templates + values
- **When to modify**:
  - Configuration options: Edit `values.yaml`
  - Deployment logic: Edit `templates/deployment.yaml`
  - RBAC: Edit `templates/rbac.yaml`
  - CRDs: Copy from `config/crd/bases/` to `crds/`

---

## How to Make Common Changes

### 1. Adding/Modifying CRD Fields

**Files to modify**:
1. `pkg/api/v1beta1/awspcaissuer_types.go` - Add field to `AWSPCAIssuerSpec` or `AWSPCAIssuerStatus`
2. Run `make manifests` - Regenerates CRDs in `config/crd/bases/`
3. Run `make generate` - Regenerates deepcopy methods
4. Copy updated CRDs from `config/crd/bases/` to `charts/aws-pca-issuer/crds/`
5. Update controller logic in `pkg/controllers/` to handle new field

**Example**: Adding a new field `timeout`:
```go
// In pkg/api/v1beta1/awspcaissuer_types.go
type AWSPCAIssuerSpec struct {
    Arn string `json:"arn,omitempty"`
    Region string `json:"region,omitempty"`
    Timeout *metav1.Duration `json:"timeout,omitempty"` // NEW FIELD
    // ... other fields
}
```

### 2. Modifying Controller Logic

**Files to modify**:
- `pkg/controllers/genericissuer_controller.go` - Issuer validation/status
- `pkg/controllers/certificaterequest_controller.go` - Certificate signing logic
- `pkg/aws/pca.go` - AWS PCA API interactions

**Testing**:
- Unit tests: `make test`
- E2E tests: `make e2etest` (requires AWS setup. For local development, rely on unit tests for validation. End to end
tests will be run as validation in PRs.)

### 3. Changing Deployment Configuration

**For Kustomize**:
- Edit `config/manager/manager.yaml` for deployment changes
- Edit `config/default/kustomization.yaml` to add/remove components
- Apply: `kubectl apply -k config/default`

**For Helm**:
- Edit `charts/aws-pca-issuer/values.yaml` for default values
- Edit `charts/aws-pca-issuer/templates/deployment.yaml` for deployment logic
- Test: `helm template charts/aws-pca-issuer`
- Install: `helm install test charts/aws-pca-issuer`

### 4. Adding New AWS PCA Features

**Files to modify**:
1. `pkg/aws/pca.go` - Add AWS SDK calls
2. `pkg/api/v1beta1/awspcaissuer_types.go` - Add configuration fields if needed
3. `pkg/controllers/certificaterequest_controller.go` - Integrate new feature
4. Update CRDs and Helm chart as needed

### 5. Modifying RBAC Permissions

**Files to modify**:
- `config/rbac/role.yaml` - Add/modify ClusterRole permissions
- `charts/aws-pca-issuer/templates/rbac.yaml` - Update Helm RBAC template
- Controller files: Add `+kubebuilder:rbac` comments above controller methods

**Example**:
```go
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
```
Then run `make manifests` to regenerate RBAC YAML.

### 6. Updating Dependencies

**Files to modify**:
- `go.mod` - Update dependency versions
- Run `go mod tidy`
- Run `go mod vendor` if using vendoring
- Test: `make test`

### 7. Changing Container Image

**Files to modify**:
- `Dockerfile` - Modify build process
- `Makefile` - Update `IMG` variable or build targets
- `charts/aws-pca-issuer/values.yaml` - Update `image.repository` and `image.tag`
- `config/manager/manager.yaml` - Update image reference (for Kustomize)

---

## Build and Test Workflow

### Development Cycle

All new code additions must have a corresponding unit test which exists within the same directory as the changed file 
and is named `<changed_file>_test`.

For instance, changes to the `pkg/controllers/certificaterequest_controller.go` file must have unit tests existing in
the `pkg/controllers/certificaterequest_controller_test.go` file. When running unit tests, ensure that coverage does not go down.
At a minimum, the coverage must remain unchanged and should increase with each change.


### Testing Hierarchy
1. **Unit tests**: `pkg/**/*_test.go` - Fast, no external dependencies
2. **E2E tests**: `e2e/e2e_test.go` - Full integration with AWS and Kubernetes
3. **Helm tests**: `e2e/helm_test.sh` - Helm chart installation validation
4. **Blog tests**: `e2e/blog_test.sh` - Validates documented workflows

---

## Important Patterns and Conventions

### 1. GenericIssuer Pattern
Both `AWSPCAIssuer` and `AWSPCAClusterIssuer` implement the `GenericIssuer` interface, allowing controllers to handle both types with the same code.

### 2. Controller Reconciliation
Controllers follow the standard Kubernetes reconciliation pattern:
- Watch resources
- Compare desired state (spec) vs actual state (status)
- Take action to converge
- Update status
- Requeue if needed

### 3. AWS Authentication Methods
Supported in order of precedence:
1. Kubernetes Secret (via `secretRef`)
2. IAM Roles for Service Accounts (IRSA)
3. IAM Roles Anywhere
4. EC2 Instance Profile
5. Environment variables

### 4. Certificate Template Mapping
The controller maps cert-manager usage types to AWS PCA templates:
- `CodeSigning` → `CodeSigningCertificate/V1`
- `ClientAuth` → `EndEntityClientAuthCertificate/V1`
- `ServerAuth` → `EndEntityServerAuthCertificate/V1`
- `OCSPSigning` → `OCSPSigningCertificate/V1`
- `ClientAuth + ServerAuth` → `EndEntityCertificate/V1`
- Everything else → `BlankEndEntityCertificate_APICSRPassthrough/V1`

### 5. Status Conditions
Issuers use standard Kubernetes conditions:
- Type: `Ready`
- Status: `True`, `False`, `Unknown`
- Reason: Short machine-readable reason
- Message: Human-readable details

---

## Configuration Files Reference

### Makefile Targets
- `make test` - Run unit tests
- `make e2etest` - Run end-to-end tests
- `make manager` - Build binary
- `make docker-build` - Build container image
- `make deploy` - Deploy to cluster
- `make manifests` - Generate CRDs and RBAC
- `make generate` - Generate deepcopy code
- `make cluster` - Create kind cluster for testing
- `make install` - Install CRDs
- `make uninstall` - Remove CRDs

### Environment Variables (for E2E tests)
- `PLUGIN_USER_NAME_OVERRIDE` - IAM user for testing
- `OIDC_S3_BUCKET_NAME` - S3 bucket for OIDC provider
- `OIDC_IAM_ROLE` - IAM role ARN for IRSA testing
- `PLUGIN_CROSS_ACCOUNT_ROLE` - Role for cross-account testing
- `TEST_KUBECONFIG_LOCATION` - Kubeconfig path

---

## Deployment Methods Comparison

| Aspect | Kustomize (config/) | Helm (charts/) |
|--------|---------------------|----------------|
| **Purpose** | Development, direct deployment | Production, distribution |
| **Command** | `kubectl apply -k config/default` | `helm install aws-pca-issuer charts/aws-pca-issuer` |
| **Customization** | Overlays and patches | Values files and `--set` flags |
| **Versioning** | Git-based | Chart version + app version |
| **Rollback** | Manual | `helm rollback` |
| **Dependencies** | None | Can declare chart dependencies |
| **When to use** | Testing, CI/CD, GitOps | User-facing releases, easy config |
| **Maintenance** | Source of truth | Manually synced from config/ |

## Additional Resources

### Documentation
- `README.md` - Main project documentation
- `CONTRIBUTING.md` - Contribution guidelines
- `docs/` - Additional documentation
- `config/examples/` - Usage examples
- `config/samples/` - Sample CRs

### External Dependencies
- **cert-manager**: Must be installed first
- **AWS PCA**: Requires configured Private CA
- **Kubernetes**: 1.13+ (varies by cert-manager version)
- **Go**: 1.23+ for development

### Related Projects
- cert-manager: https://cert-manager.io
- AWS PCA: https://aws.amazon.com/private-ca/
- Kubebuilder: https://book.kubebuilder.io

---

## Quick Reference: Where to Make Changes

| Change Type | Primary Files | Secondary Files | Commands |
|-------------|---------------|-----------------|----------|
| **Add CRD field** | `pkg/api/v1beta1/awspcaissuer_types.go` | `charts/aws-pca-issuer/crds/` | `make manifests generate` |
| **Modify controller logic** | `pkg/controllers/*.go` | `pkg/aws/pca.go` | `make test` |
| **Change AWS integration** | `pkg/aws/pca.go` | `pkg/controllers/certificaterequest_controller.go` | `make test` |
| **Update RBAC** | `config/rbac/role.yaml` | `charts/aws-pca-issuer/templates/rbac.yaml` | `make manifests` |
| **Modify deployment** | `config/manager/manager.yaml` | `charts/aws-pca-issuer/templates/deployment.yaml` | `make deploy` |
| **Change Helm defaults** | `charts/aws-pca-issuer/values.yaml` | - | `helm template` |
| **Add Helm feature** | `charts/aws-pca-issuer/templates/*.yaml` | `charts/aws-pca-issuer/values.yaml` | `helm template` |
| **Update dependencies** | `go.mod` | - | `go mod tidy` |
| **Add tests** | `pkg/**/*_test.go`, `e2e/e2e_test.go` | - | `make test`, `make e2etest` |

---

## Summary

This project follows standard Kubernetes operator patterns using Kubebuilder. The key insight is that there are **two parallel deployment methods** (Kustomize and Helm) that must be kept in sync manually. The CRDs in `config/crd/bases/` are the source of truth and should be copied to `charts/aws-pca-issuer/crds/` when updated.

When making changes:
1. Start with the Go code (`pkg/`)
2. Regenerate manifests (`make manifests generate`)
3. Update Kustomize configs (`config/`)
4. Sync changes to Helm chart (`charts/`)
5. Test both deployment methods
6. Update documentation and examples

The controller watches cert-manager CertificateRequest resources, validates them against AWSPCAIssuer/AWSPCAClusterIssuer resources, and uses the AWS PCA API to issue certificates. All AWS authentication methods are handled in `pkg/aws/pca.go`, and the certificate signing logic is in `pkg/controllers/certificaterequest_controller.go`.
