# GitHub Workflows

## PR Build Validation (`pr-build-validation.yml`)

**Purpose**: Validates that pull requests can build successfully before merge.

**Triggers**: Automatically runs on all PRs to the main branch (opened, updated, or reopened).

**What it does**:
- Sets up Go 1.24.2 environment
- Downloads dependencies
- Builds the manager binary (`make manager`)
- Runs unit tests (`make test`)

**Why it exists**: Provides immediate feedback to Q agents and developers on whether their PR changes break the build, ensuring code quality and preventing broken merges.

**Status reporting**: Results appear as PR status checks, allowing Q agents to know if their submissions are buildable.
