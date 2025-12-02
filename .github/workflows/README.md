# GitHub Workflows

## PR Build Validation (`pr-build-validation.yml`)

**Purpose**: Validates that pull requests can build successfully and pass all tests before merge.

**Triggers**: Automatically runs on all PRs to the main branch.

**What it does**:
- **Unit Tests**: Checks out code, sets up Go 1.24.2, runs `make test` (includes build validation and unit tests)
- **E2E Tests**: After unit tests pass, automatically runs comprehensive end-to-end tests on both ARM64 and x86_64 architectures using @wrichman's AWS infrastructure

**Why it exists**: Provides complete validation pipeline ensuring both unit and integration tests pass before merge. Q agents and developers get immediate feedback on code quality and functionality.

**Status reporting**: Results appear as PR status checks, allowing Q agents to know if their submissions are fully validated.

## Test Plugin (`test-plugin.yml`)

**Purpose**: Manual testing workflow for additional validation when needed.

**Triggers**: 
- Manual workflow dispatch
- PRs labeled with "safe to test" (for manual override testing)

**What it does**: Runs the same e2e test suite as the PR validation workflow, but only when manually triggered.

## E2E Test Execution (`on-safe-to-test-label.yml`)

**Purpose**: Reusable workflow that executes comprehensive end-to-end tests.

**Infrastructure**: Uses @wrichman's AWS account (783680406432) with dedicated testing roles and resources.

**Test Coverage**:
- EC2 Instance Profiles with K8s Secret authentication
- IRSA (IAM Roles for Service Accounts) with K8s Secret authentication  
- IAMRA (IAM Roles Anywhere) authentication
- Helm chart testing
- Blog example testing
