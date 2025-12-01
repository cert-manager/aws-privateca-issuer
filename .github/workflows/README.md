# GitHub Workflows

## PR Build Validation (`pr-build-validation.yml`)

**Purpose**: Validates that pull requests can build successfully before merge.

**Triggers**: Automatically runs on all PRs to the main branch.

**What it does**:
- Checks out code
- Sets up Go 1.24.2 (required by project)
- Runs `make test` (includes build validation and unit tests)

**Why it exists**: Provides immediate feedback to Q agents and developers on whether their PR changes break the build, ensuring code quality and preventing broken merges.

**Status reporting**: Results appear as PR status checks, allowing Q agents to know if their submissions are buildable.
