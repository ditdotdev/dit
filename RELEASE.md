# Datadatdat Ecosystem Release Process

This document outlines the comprehensive release process for the Datadatdat data management platform. The ecosystem consists of multiple interdependent components that must be released in a specific order to maintain compatibility.

## 🎉 What's New in v1.1.0

### Major Addition: datadatdat-remote-server Platform

**datadatdat-remote-server** is a new microservices platform that provides centralized, cloud-hosted storage for Datadatdat commits - similar to how GitHub hosts Git repositories.

**Key Features:**
- 🏗️ **5 microservices**: API Gateway, Repository Management, Ingest, Download, Worker
- 📦 **S3-compatible storage**: Uses MinIO for object storage
- 🔄 **Journal-based indexing**: Eventual consistency for high-throughput writes
- 🧪 **Comprehensive testing**: 20 E2E tests covering full workflow
- 🐳 **Docker deployment**: Full stack deployment via Docker Compose
- 🔌 **Provider integration**: Works seamlessly with d3 CLI via datadatdat-remote-go

**New Components in v1.1.0:**
1. **datadatdat-remote-go v1.1.0**: Go plugin for d3 CLI to communicate with datadatdat-remote-server
2. **datadatdat-remote 1.1.0**: Kotlin client/server providers for datadatdat-server
3. **datadatdat-remote-server v1.1.0**: 5 Docker images for the microservices platform

### Critical Changes for v1.1.0 Release

**⚠️ New Release Requirements:**
- **Clean up local development state**: Remove ALL `replace` directives from go.mod files before release
- **E2E testing requirement**: `make test-datadatdat-workflow` must pass (20/20 tests)
- **6 remote providers**: Updated count (was 5, now 6 with datadatdat-remote-go)
- **5 new Docker images**: Published to Amazon ECR (Elastic Container Registry)

**Testing Strategy:**
- E2E tests for datadatdat-remote-server are stored in `datadatdat/tests/endtoend/remotes/datadatdat/`
- Tests validate the complete integration: d3 CLI → datadatdat-server → datadatdat-remote-server
- All tests must pass both BEFORE and AFTER publishing Docker images

This document outlines the comprehensive release process for the Datadatdat data management platform. The ecosystem consists of multiple interdependent components that must be released in a specific order to maintain compatibility.

## 🚨 CRITICAL RELEASE CHECKLIST

**Before starting any release, review this checklist:**

### Pre-Release: Clean Up Local Development State
- [ ] **CRITICAL**: Remove ALL `replace` directives from go.mod files in:
  - [ ] datadatdat/go.mod
  - [ ] datadatdat-remote-go/go.mod  
  - [ ] datadatdat-remote-server/go.mod
  - [ ] nop-remote-go/go.mod
  - [ ] s3-remote-go/go.mod
  - [ ] s3web-remote-go/go.mod
  - [ ] ssh-remote-go/go.mod
- [ ] Verify all go.mod files reference published GitHub releases, not local directories

### Phase 1: Foundation - MOSTLY AUTOMATED! 🎉
- [ ] **Phase 1.1**: Tag and push `remote-sdk-go` - automation handles the rest!
  - ✅ **AUTOMATED**: Release creation with binary
  - ✅ **AUTOMATED**: PRs created in all 5 Go provider repos
  - ✅ **AUTOMATED**: Tests run in each provider
- [ ] **Phase 1.2**: Review and merge 5 provider PRs (manual review required)
- [ ] **Phase 1.2**: Tag and push all 5 providers - releases publish automatically
- [ ] **Phase 1.3**: ⚠️ **REQUIRED** - Release `remote-sdk` (Kotlin/Maven) version 1.1.0 BEFORE Phase 2

### Phase 2: Kotlin Providers  
- [ ] **Phase 2**: Release 6 Kotlin remote providers with version 1.1.0 (NO 'v' prefix)
- [ ] **Phase 2**: Verify datadatdat-remote publishes BOTH client and server artifacts to Maven

### Phase 3-6: Core Components
- [ ] **Phase 3**: Release `datadatdat-client-go` v1.1.0 (if needed)
- [ ] **Phase 4**: Release `datadatdat-docker-proxy` v1.1.0 (⚠️ REQUIRED before Phase 5)
- [ ] **Phase 5**: Release datadatdat-server v1.1.0 (⚠️ REQUIRED before Phase 6 - CLI E2E tests need this)
- [ ] **Phase 6**: Remove replace directives from datadatdat/go.mod
- [ ] **Phase 6**: Update datadatdat CLI dependencies to use NEW remote provider versions
- [ ] **Phase 6**: Verify dependency alignment: `go mod graph | grep datadatdat | grep remote-sdk-go`
- [ ] **Phase 6**: Release datadatdat CLI with aligned dependencies (E2E tests will use Phase 5 server image)

### Phase 7: Remote Server Platform
- [ ] **Phase 7.1**: Remove replace directives from datadatdat-remote-server/go.mod
- [ ] **Phase 7.2**: Run local E2E tests - `make test-datadatdat-workflow` must pass
- [ ] **Phase 7.3**: Release datadatdat-remote-server v1.1.0 (6 Docker images)
- [ ] **Phase 7.4**: Verify all 6 Docker images published to DockerHub
- [ ] **Phase 7.5**: Run E2E tests against released images

### Post-Release: Validation
- [ ] **Post-Release**: Validate entire ecosystem has consistent dependency versions
- [ ] **Post-Release**: Full E2E test suite passes: `make e2e`
- [ ] **Post-Release**: datadatdat remote workflow tests pass: `make test-datadatdat-workflow`

**⚠️ Phase 1.2 and replace directive cleanup are CRITICAL - missing these causes version conflicts!**

## �️ New Architecture: datadatdat-remote-server ("GitHub for Data")

### What is datadatdat-remote-server?

**datadatdat-remote-server** is to Datadatdat (d3) what **GitHub is to Git**:
- Just as you can use git with any SSH server (basic) OR use GitHub (web UI, orgs, PRs, collaboration)
- Users can use d3 with S3/SSH directly (basic) OR use datadatdat-remote-server (web UI, orgs, auth, APIs)

### Architecture Overview

**Microservices Platform (5 Docker Images):**
1. **api-gateway**: Envoy-based API gateway (routing, auth, rate limiting)
2. **api-repo-manifest**: Repository and manifest management
3. **api-ingest**: Upload/commit ingestion with multipart support
4. **api-download**: Download and streaming of commit archives
5. **worker**: Background processing for async tasks
5. **worker**: Background processing (index refresh, cleanup, metrics)
6. **datadatdat-provider-http**: gRPC provider plugin for d3 CLI integration

**Supporting Services:**
- MinIO (S3-compatible object storage)
- PostgreSQL (metadata and user management)
- Grafana + Prometheus (monitoring)
- OpenTelemetry Collector (distributed tracing)

### End-to-End Testing Strategy

**Critical for Release: E2E Tests in datadatdat Repository**

The E2E tests for datadatdat-remote-server are stored in the **datadatdat** repository (NOT in datadatdat-remote-server):
- Location: `datadatdat/tests/endtoend/remotes/datadatdat/datadatdatWorkflowTests.yml`
- Run via: `make test-datadatdat-workflow` (from datadatdat directory)
- Tests: 20 comprehensive workflow tests covering push/pull/checkout/delete operations

**Why tests are in datadatdat repo:**
- Tests the full integration: d3 CLI → datadatdat-server → datadatdat-remote-server
- Validates the complete user workflow from CLI perspective
- Ensures compatibility between all components
- Follows the pattern of other remote tests (s3, ssh, s3web)

**Release Requirement:**
- ALL 20 tests MUST pass before releasing datadatdat-remote-server
- Tests must pass BOTH before and after publishing Docker images
- Validates that released images work correctly in real-world scenarios

### Provider Architecture

**Three-Layer Provider System:**

1. **datadatdat-remote-go** (Go plugin for d3 CLI)
   - Provides URL parsing: `http://localhost:8080/org/repo`
   - gRPC plugin protocol for d3 CLI integration
   - Implements remote-sdk-go interface
   - Published as GitHub release with Go binary

2. **datadatdat-remote** (Kotlin Maven artifacts)
   - **client artifact**: `datadatdat-remote-client:1.1.0` (URL parsing, validation)
   - **server artifact**: `datadatdat-remote-server:1.1.0` (HTTP operations, push/pull logic)
   - Both registered via ServiceLoader in datadatdat-server
   - Published to S3 Maven repository

3. **datadatdat-provider-http** (Go gRPC service)
   - Runs as Docker container in datadatdat-remote-server stack
   - Bridges d3 CLI gRPC calls to HTTP REST APIs
   - Handles authentication and request routing

**Critical Integration Points:**
- d3 CLI loads datadatdat-remote-go via go-plugin
- datadatdat-server loads datadatdat-remote Kotlin providers via ServiceLoader
- All components must be at compatible versions for E2E tests to pass

## 🎯 Current v1.1.0 Release Progress

### Completed Work (October 2025)
- [x] **datadatdat-remote-server** - Fully implemented and tested
  - All 6 microservices operational
  - E2E tests: 20/20 passing (100%)
  - Docker Compose deployment working
  - GitHub Actions CI/CD configured
  
- [x] **datadatdat-remote-go v1.0.0** - Published
  - Full HTTP client implementation
  - 61 tests, 96% coverage
  - Integrated with d3 CLI
  
- [x] **datadatdat-remote v1.0.0** - Published to Maven
  - Client and server artifacts published
  - ServiceLoader registration working
  - Integrated with datadatdat-server

### Ready for v1.1.0 Release
- [ ] **Phase 1**: Update and release remote-sdk-go v1.1.0
- [ ] **Phase 2**: Update and release all Go remote providers v1.1.0
- [ ] **Phase 3**: Update and release all Kotlin remote providers 1.1.0
- [ ] **Phase 4**: Release datadatdat-docker-proxy v1.1.0
- [ ] **Phase 5**: Release datadatdat-server v1.1.0 (⚠️ BEFORE CLI)
- [ ] **Phase 6**: Release datadatdat CLI v1.1.0 (E2E tests need server)
- [ ] **Phase 7**: Release datadatdat-remote-server v1.1.0 (6 Docker images)

**Last Updated**: October 20, 2025 - Ready for v1.1.0 release

## Release Dependencies and Order

### Component Dependency Graph
```
remote-sdk-go (foundation)
    ↓
[s3-remote-go, ssh-remote-go, s3web-remote-go, nop-remote-go, datadatdat-remote-go] (remote providers)
    ↓
datadatdat-client-go (auto-generated from datadatdat-server OpenAPI spec)
    ↓
datadatdat-docker-proxy (binary downloaded by datadatdat-server during Docker build)
    ↓
datadatdat-server (Docker container with ZFS + PostgreSQL)
    ↓
datadatdat (CLI - depends on all remote providers and client)
    ↓
datadatdat-remote-server (Microservices platform - "GitHub for Data")
```

### Release Order (Critical)
1. **command-executor** (if changed) - Foundation library, must release BEFORE remote-sdk
2. **remote-sdk-go** - Foundation SDK for all remote providers
3. **Remote providers** (can be done in parallel):
   - s3-remote-go
   - ssh-remote-go  
   - s3web-remote-go
   - nop-remote-go
   - **datadatdat-remote-go** - NEW: Provider for datadatdat-remote-server
4. **remote-sdk** (Kotlin/Maven) - Foundation for Kotlin providers (depends on command-executor)
5. **Kotlin remote providers** (can be done in parallel):
   - s3-remote
   - ssh-remote (depends on command-executor)
   - s3web-remote
   - nop-remote
   - delphix-remote (depends on command-executor)
   - **datadatdat-remote** - NEW: Server-side provider for datadatdat-remote-server
6. **datadatdat-client-go** - Auto-generated Go client
7. **datadatdat-docker-proxy** - 🚨 CRITICAL: Must be BEFORE datadatdat-server (binary downloaded during Docker build)
8. **datadatdat-server** - Docker container (publishes to DockerHub, downloads docker-volume-proxy from S3)
9. **datadatdat** - Main CLI (depends on all above, runs E2E tests against datadatdat-server)
10. **datadatdat-remote-server** - NEW: Microservices platform (publishes 6 Docker images)

### Supporting Components (Independent)
- **plugin-launcher** - Kotlin library with Go tests; can be released independently (no current dependencies)
- **zfs-builder**, **zfs-releases** - ZFS infrastructure, independent
- **Testing** - Now uses BATS (Bash Automated Testing System) instead of previous custom testing solution

## Version Strategy

### Target Version for v1.1.0 Release
**ALL components will be updated to v1.1.0 for this major release:**
- **datadatdat**: v1.1.0 (main CLI)
- **datadatdat-server**: v1.1.0 (Docker container `datadatdat/datadatdat:1.1.0`)
- **datadatdat-remote-server**: v1.1.0 (6 Docker images: api-gateway, api-repo-manifest, api-ingest, api-download, worker, datadatdat-provider-http)
- **datadatdat-client-go**: v1.1.0 (auto-generated client)
- **remote-sdk-go**: v1.1.0 (foundation SDK)
- **All Go remote providers**: v1.1.0 (including new datadatdat-remote-go)
- **All Kotlin components**: 1.1.0 (Maven artifacts - NO 'v' prefix, including new datadatdat-remote)

### CRITICAL: Maven Versioning Requirements
⚠️ **Kotlin/Maven repositories MUST use semantic versioning WITHOUT the 'v' prefix:**
- ✅ **CORRECT**: `1.0.0` for Maven artifacts
- ❌ **WRONG**: `v1.0.0` (causes S3 Maven publishing failures)

**Affected repositories:**
- remote-sdk, command-executor, plugin-launcher
- s3-remote, ssh-remote, s3web-remote, nop-remote, delphix-remote

**Publishing pattern:**
```bash
# For Kotlin/Maven repos - NO 'v' prefix
./gradlew publish -Pversion=1.0.0

# For Go repos - WITH 'v' prefix for Git tags
git tag v1.0.0
```

### Previous Versioning Scheme (Reference)
- **datadatdat**: v0.5.x (main CLI)
- **datadatdat-server**: v0.8.x (Docker container)
- **datadatdat-client-go**: v0.1.x (auto-generated client)
- **remote-sdk-go**: v0.2.x (foundation SDK)
- **Remote providers**: v0.2.x (aligned with SDK)

### Versioning Rules
1. **Semantic Versioning**: All components use semver (vMAJOR.MINOR.PATCH)
2. **Dependency Alignment**: Remote providers should align with remote-sdk-go versions
3. **CLI Independence**: Datadatdat CLI version advances independently but must reference compatible dependency versions
4. **Server Alignment**: datadatdat-server version should generally align with datadatdat CLI for major releases

## Complete Datadatdat Release Process - Step by Step

### Pre-Release Phase (1-2 days before)

#### 0. Critical: Clean Up Local Development Dependencies
```bash
# ⚠️ MUST BE DONE FIRST - Remove all local replace directives
# During development, we use local replace directives for fast iteration
# For release, ALL dependencies must reference published GitHub versions

# List of repositories with go.mod files that may have replace directives:
REPOS=(
  "datadatdat"
  "datadatdat-remote-go"
  "datadatdat-remote-server"
  "nop-remote-go"
  "s3-remote-go"
  "s3web-remote-go"
  "ssh-remote-go"
)

# For each repo, check for replace directives
for repo in "${REPOS[@]}"; do
  echo "Checking /c/dev/$repo/go.mod"
  cd /c/dev/$repo
  
  # Show any replace directives
  grep "replace" go.mod || echo "✅ No replace directives found"
  
  # If replace directives exist, remove them manually
  # Then run: go mod tidy
done

# Example of what to remove from go.mod:
# replace github.com/datadatdat/datadatdat-remote-go => ../datadatdat-remote-go
# replace github.com/datadatdat/remote-sdk-go => ../remote-sdk-go

# After removing, ensure dependencies reference published versions:
# require github.com/datadatdat/remote-sdk-go v1.1.0
# require github.com/datadatdat/datadatdat-remote-go v1.1.0
```

**⚠️ Why This Is Critical:**
- Local replace directives work only on your machine
- Published releases must use public GitHub versions
- Users installing d3 CLI can't access your local directories
- Build failures occur if replace directives reference non-existent paths

#### 1. Pre-Release Planning
```bash
# Determine version increments for all components
# For this release: v1.1.0 (Go) and 1.1.0 (Kotlin/Maven)
# Check for breaking changes that require major version bumps
# Coordinate with team on release timing
```

#### 2. Documentation Review
```bash
cd /c/dev/datadatdat-data.github.io
# Review and update documentation for new features
# Prepare release notes and changelog entries
```

#### 3. OpenAPI Specification Sync
```bash
cd /c/dev/datadatdat-server
# Ensure OpenAPI spec (openapi/datadatdat.yml) reflects all server changes
# This will trigger datadatdat-client-go regeneration in next phase
```

### Release Phase Day

#### Phase 1: Foundation Components (Go Modules)

##### 1.1 Release remote-sdk-go - FULLY AUTOMATED! 🚀
```bash
cd /c/dev/remote-sdk-go

# Just tag and push - everything else is automatic!
export NEW_SDK_VERSION="v1.3.0"
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION

# 🎉 What happens automatically (all in ONE workflow run):
# 1. ✅ Builds and tests the SDK binary
# 2. ✅ Creates and publishes GitHub release immediately
# 3. ✅ Updates go.mod in ALL 5 provider repos
# 4. ✅ Runs tests in each provider repo
# 5. ✅ Creates PRs in all 5 provider repos

# Monitor the workflow
gh run watch

# After ~2-3 minutes, check for PRs
gh pr list --repo datadatdat/s3-remote-go
gh pr list --repo datadatdat/ssh-remote-go
gh pr list --repo datadatdat/s3web-remote-go
gh pr list --repo datadatdat/nop-remote-go
gh pr list --repo datadatdat/datadatdat-remote-go

# Each PR will have title: "Update remote-sdk-go to v1.3.0"

# Check PR status and CI results (recommended: wait 30s for checks to start)
for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  echo "=== Checking $repo ==="
  gh pr view auto-update-sdk-$NEW_SDK_VERSION --repo datadatdat/$repo --json number,title,checks
done

# Or check a specific PR by number
gh pr checks <PR_NUMBER> --repo datadatdat/<REPO_NAME>
```

**✅ FULLY AUTOMATED (v1.3.0+):** One tag push triggers SDK release + PR creation in all 5 providers!

##### 1.2 Merge Provider PRs and Release
**What's automated:** PRs created, go.mod updated, tests run, everything built  
**Your job:** Review and merge (5 PRs), then tag each provider (5 tags)

```bash
# Check all PRs are passing (wait for CI to complete if needed)
for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  echo "=== $repo ==="
  gh pr view auto-update-sdk-$NEW_SDK_VERSION --repo datadatdat/$repo --json number,title,checks --jq '.checks[] | "\(.name): \(.conclusion)"'
done

# Review and merge the PRs (automated tests already passed)
# Replace <PR_NUMBER> with actual PR numbers from gh pr list
gh pr merge <PR_NUMBER> --repo datadatdat/s3-remote-go --squash --delete-branch
gh pr merge <PR_NUMBER> --repo datadatdat/ssh-remote-go --squash --delete-branch
gh pr merge <PR_NUMBER> --repo datadatdat/s3web-remote-go --squash --delete-branch
gh pr merge <PR_NUMBER> --repo datadatdat/nop-remote-go --squash --delete-branch
gh pr merge <PR_NUMBER> --repo datadatdat/datadatdat-remote-go --squash --delete-branch

# After merging, tag each provider (triggers automatic release)
export NEW_PROVIDER_VERSION="v1.3.0"
for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  cd /c/dev/$repo
  git pull origin master
  git tag $NEW_PROVIDER_VERSION
  git push origin $NEW_PROVIDER_VERSION
  echo "✅ Tagged $repo at $NEW_PROVIDER_VERSION"
done

# GitHub Actions automatically build, test, and publish releases
# Releases are published immediately (no draft step)
```

**✅ VALIDATION: After providers release, verify version alignment:**
```bash
cd /c/dev/datadatdat
go get github.com/datadatdat/s3-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/ssh-remote-go@$NEW_PROVIDER_VERSION  
go get github.com/datadatdat/s3web-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/nop-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/datadatdat-remote-go@$NEW_PROVIDER_VERSION
go mod tidy
go mod graph | grep datadatdat | grep remote-sdk-go
# ALL providers should show the SAME remote-sdk-go version
```

##### 1.3 Release remote-sdk (Kotlin/Maven) - REQUIRED BEFORE KOTLIN PROVIDERS
**⚠️ WARNING: Kotlin providers depend on this - must be released BEFORE Phase 2!**

```bash
cd /c/dev/remote-sdk

# Test build locally
./gradlew build test

# Tag and push (triggers automated Maven publishing)
export NEW_SDK_VERSION="1.1.0"  # Use 1.1.0 for this release (NO 'v' prefix for Kotlin/Maven!)
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION

# GitHub Action automatically:
# - Builds and tests the JAR
# - Publishes to S3 Maven bucket (datadatdat-maven)
# - Creates GitHub draft release

# Wait for GitHub Action to complete
gh run list --workflow=release.yml --limit 3
# Look for ✓ status

# CRITICAL: Verify artifact was published to S3 Maven bucket
aws s3 ls s3://datadatdat-maven/com/datadatdat/remote-sdk/1.1.0/
# Should show files like:
#   remote-sdk-1.1.0.jar
#   remote-sdk-1.1.0.pom
#   remote-sdk-1.1.0-sources.jar
#   remote-sdk-1.1.0-javadoc.jar

# If artifact is missing, Kotlin providers will fail to build with 403 Forbidden error
```

**✅ VALIDATION: Verify remote-sdk 1.1.0 is published to Maven before continuing to Phase 2**

##### 1.4 Update Provider build.gradle.kts Files - CRITICAL BEFORE TAGGING
**🚨 NEW STEP (v1.4.0+): Update dependency versions BEFORE releasing providers**

**Why This Matters:**
- We release `remote-sdk` first as 1.4.0
- Provider `build.gradle.kts` files still reference `remote-sdk:1.3.0`
- If we tag providers NOW, they publish with OLD dependency versions
- Published Maven artifacts will be INCORRECT
- Must update source files BEFORE tagging

```bash
# For each Kotlin provider, update build.gradle.kts to use NEW versions
# This is what we missed in the initial v1.4.0 release!

# Example for datadatdat-remote:
cd /c/dev/datadatdat-remote
# Update server/build.gradle.kts: remote-sdk:1.3.0 → remote-sdk:1.4.0
# Update client/build.gradle.kts: remote-sdk:1.3.0 → remote-sdk:1.4.0

# ⚠️ IMPORTANT: Update ALL 6 Kotlin provider repos:
# - datadatdat-remote (server + client)
# - s3-remote (server + client)
# - ssh-remote (server + client) - also has command-executor dependency
# - s3web-remote (server + client)
# - nop-remote (server + client)
# - delphix-remote (server + client) - also has command-executor dependency

# AUTOMATED UPDATE SCRIPT (recommended):
cd /c/dev && for repo in datadatdat-remote s3-remote ssh-remote s3web-remote nop-remote delphix-remote; do
  echo "=== Updating $repo ==="
  cd /c/dev/$repo
  
  # Update server build.gradle.kts (remote-sdk dependency)
  sed -i 's/remote-sdk:1\.3\.0/remote-sdk:1.4.0/g' server/build.gradle.kts
  
  # Update client build.gradle.kts (remote-sdk dependency)
  sed -i 's/remote-sdk:1\.3\.0/remote-sdk:1.4.0/g' client/build.gradle.kts
  
  # For ssh-remote and delphix-remote, also update command-executor (if changed)
  if [[ "$repo" == "ssh-remote" || "$repo" == "delphix-remote" ]]; then
    sed -i 's/command-executor:1\.3\.0/command-executor:1.4.0/g' server/build.gradle.kts
  fi
  
  # Commit the changes
  git add server/build.gradle.kts client/build.gradle.kts
  git commit -m "Update dependencies to remote-sdk:1.4.0"
  git push origin master
  
  echo "✓ $repo updated"
done

# ✅ VALIDATION: Verify all build.gradle.kts files show correct versions
for repo in datadatdat-remote s3-remote ssh-remote s3web-remote nop-remote delphix-remote; do
  echo "=== $repo ==="
  grep "com.datadatdat:remote-sdk:" /c/dev/$repo/server/build.gradle.kts
done
# All should show remote-sdk:1.4.0

# NOW you can proceed to Phase 2 and tag the providers
```

**✅ VALIDATION: All provider build.gradle.kts files updated and committed BEFORE tagging**

##### 1.3 Release command-executor (Foundation Library - REQUIRED if changed)

**CRITICAL:** command-executor is a **dependency** of multiple components. Must be released BEFORE remote-sdk if there are changes.

**Dependencies:** remote-sdk, datadatdat-server, ssh-remote, delphix-remote

```bash
cd /c/dev/command-executor

# Check for changes since last release
git log --oneline <LAST_TAG>..HEAD

# If NO changes: Skip to Phase 1.4 (remote-sdk)
# If changes exist: MUST release before remote-sdk

# Build and test
./gradlew build test

# Tag and push (triggers automated Maven publishing to S3)
git tag 1.4.0  # Note: NO 'v' prefix for Maven artifacts
git push origin 1.4.0

# Wait for GitHub Actions to publish
gh run watch

# CRITICAL: Verify published to S3 Maven bucket before continuing
aws s3 ls s3://datadatdat-maven/com/datadatdat/command-executor/1.4.0/
# Should show: command-executor-1.4.0.jar, .pom, etc.

# ⚠️ IF COMMAND-EXECUTOR WAS RELEASED:
# Update dependency version in these files BEFORE their releases:
# - remote-sdk/build.gradle.kts (line 40)
# - datadatdat-server/server/build.gradle.kts (line 43)
# - ssh-remote/server/build.gradle.kts (line 24)
# - delphix-remote/server/build.gradle.kts (line 25)
```

##### 1.4 Release remote-sdk (Kotlin/Maven)

#### Phase 2: Kotlin Remote Providers (Maven JARs) - Parallel Process

For each Kotlin remote (s3-remote, ssh-remote, s3web-remote, nop-remote, delphix-remote, datadatdat-remote):

```bash
cd /c/dev/s3-remote  # Repeat for each Kotlin remote

# Update remote-sdk dependency if needed
# (sed command or manual edit of server/build.gradle.kts)

# Test build locally
./gradlew build test

# Commit changes if needed
git add server/build.gradle.kts
git commit -m "Update remote-sdk to 1.1.0"
git push origin master

# Create git tag (triggers automated Maven publishing)
git tag 1.1.0
git push origin 1.1.0

# Wait for GitHub Action to complete
gh run list --workflow=release.yml --limit 3
# Look for ✓ status

# CRITICAL: Verify artifact was published to S3 Maven bucket
aws s3 ls s3://datadatdat-maven/com/datadatdat/s3-remote-server/1.1.0/
# Should show files like:
#   s3-remote-server-1.1.0.jar
#   s3-remote-server-1.1.0.pom
#   s3-remote-server-1.1.0-sources.jar

# If artifact is missing, DO NOT continue - debug the GitHub Actions workflow
```

**⚠️ IMPORTANT for datadatdat-remote:**
The datadatdat-remote repository has TWO Maven artifacts:
- `datadatdat-remote-client:1.1.0` - Client-side provider for d3 CLI
- `datadatdat-remote-server:1.1.0` - Server-side provider for datadatdat-server

Both artifacts are published automatically when the tag is pushed.

#### Phase 3: Auto-Generated Client

##### 3.1 Release datadatdat-client-go
```bash
cd /c/dev/datadatdat-client-go

# If OpenAPI spec changed, regenerate client first
# (This may be automated or require manual trigger)

# Tag and push release
git tag v1.1.0
git push origin v1.1.0

# Wait for GitHub Action to complete
gh run list --workflow=release.yml --limit 3
# Look for ✓ status

# Verify draft release created
gh release list --limit 3
gh release view v1.1.0

# If draft, publish the release
gh release edit v1.1.0 --draft=false --latest

# Verify published
gh release list --limit 3
# Should show "v1.1.0  Latest"
```

**Note:** datadatdat-client-go releases are automatically published (not draft) because it's a Go library with no binaries to verify. The workflow creates a draft, runs tests, then immediately publishes. Other Go repos (remote-sdk-go, providers) stay in draft until manually published because they build binaries that should be verified first.


#### Phase 4: Main CLI Release

##### 4.1 Update d3 CLI Dependencies
```bash
cd /c/dev/datadatdat

# CRITICAL: Remove ALL local replace directives from go.mod before release
# Check for replace directives
grep "replace" go.mod

# Remove all replace directives and update to released versions
# Edit go.mod to remove lines like:
# replace github.com/datadatdat/datadatdat-remote-go => ../datadatdat-remote-go
# replace github.com/datadatdat/remote-sdk-go => ../remote-sdk-go
# etc.

# Update all dependencies to latest released versions
go get github.com/datadatdat/nop-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go get github.com/datadatdat/s3-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/s3web-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/ssh-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/datadatdat-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/datadatdat-client-go@$NEW_CLIENT_VERSION
go mod tidy

# Verify no version conflicts
go mod graph | grep datadatdat | grep remote-sdk-go
# All providers should use same remote-sdk-go version
```

##### 4.2 Commit, Tag, and Push - EVERYTHING ELSE IS AUTOMATED
**✅ AUTOMATED (v1.3.0+):** Build, test, release creation, and publishing all happen automatically via GitHub Actions

```bash
export VERSION="v1.3.0"  # Set your version

# Stage dependency updates
git add go.mod go.sum

# Commit with version message
git commit -m "Release $VERSION: Update all dependencies"

# Push commits first
git push origin master

# Create and push tag - THIS TRIGGERS THE AUTOMATION
git tag $VERSION
git push origin $VERSION

# Now sit back and let GitHub Actions do the work:
# 1. ✅ Builds 5 cross-platform binaries (darwin/linux/windows, amd64/arm64)
# 2. ✅ Runs full E2E test suite with BATS + ZFS
# 3. ✅ Creates GitHub release with all artifacts
# 4. ✅ Publishes release (NO DRAFT - goes live immediately)

# Monitor the workflow
gh run watch

# Or check status
gh run list --workflow=release.yml --limit 3
```

**What the `release.yml` workflow does:**
- Checks out code at the tag
- Configures Go with GO_MODULES_TOKEN for private repos
- Runs `make release` to build all platforms
- Installs BATS and ZFS
- Runs complete E2E test suite
- Creates release with all binary artifacts
- Publishes immediately (no manual review needed)

**If release fails:**
```bash
# Check the workflow logs
gh run list --workflow=release.yml --limit 3
gh run view <RUN_ID> --log-failed

# If tests fail, delete tag and fix issues
git tag -d $VERSION 
git push origin --delete $VERSION

# Fix issues, then retry with patch version
export VERSION="v1.3.1"
# ... repeat process
- Added datadatdat-remote-server integration (\"GitHub for Data\")
- All 5 remote providers now fully functional
- Improved error handling and test reliability

### ✅ Testing
- 100+ E2E tests passing
- Full integration testing completed

### 📦 Artifacts
All cross-platform binaries included."
```

**⚠️ CRITICAL POST-RELEASE VALIDATION:**
```bash
# After releasing CLI, verify ENTIRE ecosystem has aligned dependencies
go mod graph | grep datadatdat | grep remote-sdk-go
# Should show ALL components using the SAME remote-sdk-go version
# If you see version conflicts, you MUST create a patch release to fix alignment
```

#### Phase 4.5: Docker Volume Proxy Release

##### 4.5.1 Release datadatdat-docker-proxy
**🚨 CRITICAL: Must be released BEFORE datadatdat-server**

**Why:** datadatdat-server's Dockerfile downloads `docker-volume-proxy` binary from S3 during build:
```dockerfile
RUN curl -fssL https://datadatdat-maven.s3.amazonaws.com/datadatdat-docker-proxy/docker-volume-proxy -o /datadatdat/docker-volume-proxy
```

**When to release:**
- After datadatdat-client-go is released (Phase 3)
- Before datadatdat-server is released (Phase 5)
- If datadatdat-docker-proxy has ANY changes since last release

```bash
cd /c/dev/datadatdat-docker-proxy

# Check current dependencies
grep "datadatdat-client-go" go.mod
# Should show: github.com/datadatdat/datadatdat-client-go v1.4.0

# If client version is outdated, update it
go get github.com/datadatdat/datadatdat-client-go@v1.4.0
go mod tidy

# Commit if there are changes
git add go.mod go.sum
git commit -m "Update datadatdat-client-go to v1.4.0"
git push origin master

# Tag and push (triggers automated build and S3 upload)
git tag v1.4.0
git push origin v1.4.0

# GitHub Action automatically:
# - Builds docker-volume-proxy binary
# - Uploads to S3: s3://datadatdat-maven/datadatdat-docker-proxy/docker-volume-proxy
# - Creates GitHub release

# Wait for GitHub Action to complete
gh run list --workflow=release.yml --limit 3
# Look for ✓ status

# CRITICAL: Verify binary was uploaded to S3
aws s3 ls s3://datadatdat-maven/datadatdat-docker-proxy/
# Should show: docker-volume-proxy

# Test the binary is downloadable
curl -fsSL https://datadatdat-maven.s3.amazonaws.com/datadatdat-docker-proxy/docker-volume-proxy --head
# Should return 200 OK
```

**✅ VALIDATION: Verify docker-volume-proxy is in S3 before releasing datadatdat-server**

#### Phase 5: Docker Container Release

**⚠️ BEFORE TAGGING datadatdat-server**: Update all dependencies in datadatdat-server to match the new release version.

**Files to Update**:
1. `server/build.gradle.kts` - All `com.datadatdat:*` Maven dependencies  
2. `go.mod` - `datadatdat-client-go` version

**Steps**:
```bash
cd /c/dev/datadatdat-server

# Check current versions
grep "com.datadatdat:" server/build.gradle.kts
grep "datadatdat-client-go" go.mod

# Update server/build.gradle.kts - change ALL old versions to new version:
# - com.datadatdat:command-executor:$VERSION
# - com.datadatdat:remote-sdk:$VERSION
# - com.datadatdat:datadatdat-remote-client:$VERSION
# - com.datadatdat:datadatdat-remote-server:$VERSION
# - com.datadatdat:nop-remote-server:$VERSION
# - com.datadatdat:ssh-remote-server:$VERSION
# - com.datadatdat:s3-remote-server:$VERSION
# - com.datadatdat:s3web-remote-server:$VERSION

# Update go.mod - change datadatdat-client-go to v$VERSION
go mod tidy

# Commit and push BEFORE tagging
git add server/build.gradle.kts go.mod go.sum
git commit -m "Update dependencies to $VERSION"
git push origin master
```

**Verification**:
```bash
# Verify all 8 Kotlin dependencies updated
grep "com.datadatdat:.*:$VERSION" server/build.gradle.kts | wc -l  # Should be 8
# Verify Go dependency updated  
grep "datadatdat-client-go v$VERSION" go.mod
```

---

**Now proceed with datadatdat-server release**:

##### 5.1 Release datadatdat-server
```bash
cd /c/dev/datadatdat-server

# Ensure compatibility with new d3 CLI version
# Update any version references if needed

# Create tag - this triggers automated publishing
export NEW_SERVER_VERSION="v1.1.0"  # Use v1.1.0 for this release
git tag $NEW_SERVER_VERSION
git push origin $NEW_SERVER_VERSION

# GitHub Action automatically:
# - Runs full test suite including E2E tests
# - Builds multi-arch Docker image (linux/amd64, linux/arm64)  
# - Publishes to DockerHub as datadatdat/datadatdat:$NEW_SERVER_VERSION
# - Tags and publishes datadatdat/datadatdat:latest
# - Creates GitHub draft release
```

#### Phase 6: Datadatdat Remote Server Release ("GitHub for Data" Platform)

##### 6.1 Prepare for Release
```bash
cd /c/dev/datadatdat-remote-server

# CRITICAL: Remove local replace directives from go.mod
# The go.mod file should NOT have any local replace directives for release
# All dependencies must point to published GitHub releases

# Check current go.mod for replace directives
grep "replace" go.mod

# If any local replaces exist, remove them and update to use published versions
# Example: Should have github.com/datadatdat/remote-sdk-go v1.1.0
# NOT: replace github.com/datadatdat/remote-sdk-go => ../remote-sdk-go

# Update dependencies to released versions
go get github.com/datadatdat/remote-sdk-go@v1.1.0
go mod tidy

# Commit the updated go.mod
git add go.mod go.sum
git commit -m "Update dependencies for v1.1.0 release"
git push origin master
```

##### 6.2 Run Local End-to-End Tests
```bash
cd /c/dev/datadatdat-remote-server

# Start the full stack locally
docker-compose -f deploy/compose/docker-compose.yml up -d

# Wait for all services to be healthy
sleep 30

# Run integration tests
make test

# CRITICAL: Run E2E tests from datadatdat CLI
# These tests are stored in the datadatdat repository
cd /c/dev/datadatdat
make test-datadatdat-workflow

# Expected output: All tests should pass (20/20)
# If tests fail, DO NOT proceed with release until issues are resolved

# Cleanup
cd /c/dev/datadatdat-remote-server
docker-compose -f deploy/compose/docker-compose.yml down
```

##### 6.3 Release datadatdat-remote-server to Amazon ECR
```bash
cd /c/dev/datadatdat-remote-server

# IMPORTANT: Ensure datadatdat v1.1.0 is already released (Phase 4)
# The workflow checks out both repos at matching version for E2E testing

# Create tag - this triggers automated build, test, and publish
export NEW_VERSION="v1.1.0"  # Use v1.1.0 for this release
git tag $NEW_VERSION
git push origin $NEW_VERSION

# GitHub Action automatically:
# Job 1: Build
#   - Builds 8 Docker images (linux/amd64):
#     * $ECR_REGISTRY/datadatdat/api-gateway:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/api-repo-manifest:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/api-ingest:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/api-download:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/worker:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/auth-server:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/web:v1.1.0 and :latest
#     * $ECR_REGISTRY/datadatdat/datadatdat-repo-web:v1.1.0 and :latest
#   - Saves images as artifacts
#
# Job 2: E2E Testing (CRITICAL - tests before publish!)
#   - Checks out datadatdat-remote-server at v1.1.0
#   - Checks out datadatdat at v1.1.0 (requires GO_MODULES_TOKEN secret)
#   - Loads built Docker images
#   - Builds datadatdat CLI
#   - Installs OpenZFS
#   - Starts datadatdat-remote-server stack with built images
#   - Runs make test-datadatdat-workflow (20 E2E tests)
#   - Tests must pass 100% before proceeding
#
# Job 3: Publish to ECR (only if tests pass)
#   - Authenticates to Amazon ECR using AWS credentials
#   - Pushes all version tags (v1.1.0)
#   - Pushes all latest tags
#   - Creates GitHub draft release with deployment instructions
```

##### 6.4 Authenticate to Amazon ECR and Verify Release
```bash
# Authenticate to Amazon Elastic Container Registry
# You need AWS credentials with ECR permissions (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
export ECR_REGISTRY=$(aws ecr describe-repositories --region us-west-2 --repository-names datadatdat/api-gateway --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Pull all published images from ECR (private registry)
docker pull $ECR_REGISTRY/datadatdat/api-gateway:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-repo-manifest:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-ingest:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-download:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/worker:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/auth-server:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/web:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/datadatdat-repo-web:v1.1.0

# Verify latest tags are also available
docker pull $ECR_REGISTRY/datadatdat/api-gateway:latest

# Check GitHub release
gh release view v1.1.0 --repo datadatdat/datadatdat-remote-server

# Verify all 8 images are listed in the release notes
```

##### 6.5 Deploy and Test with Amazon ECR Images
```bash
# CRITICAL: Verify the released images work correctly

cd /c/dev/datadatdat-remote-server/deploy/compose

# Authenticate to Amazon ECR (if not already done)
export ECR_REGISTRY=$(aws ecr describe-repositories --region us-west-2 --repository-names datadatdat/api-gateway --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Create .env file to use ECR and specific version
cat > .env << EOF
REGISTRY=$ECR_REGISTRY/
VERSION=v1.1.0
EOF

# Pull the released images
docker-compose pull

# Start the stack
docker-compose up -d

# Wait for services to be healthy
sleep 30

# Check service status
docker-compose ps

# Run E2E tests from datadatdat CLI against released images
cd /c/dev/datadatdat
make test-datadatdat-workflow

# Expected: All tests pass (20/20)
# If tests fail with released images, investigate immediately

# Cleanup
cd /c/dev/datadatdat-remote-server/deploy/compose
docker-compose down -v

# Remove .env file to go back to local development mode
rm .env
```

##### 6.6 Deployment Options Reference

**Option 1: Build from Source (Development)**
```bash
cd /c/dev/datadatdat-remote-server/deploy/compose

# No .env file needed - uses build contexts by default
docker-compose build
docker-compose up -d
```

**Option 2: Pull from Amazon ECR - Specific Version (Production)**
```bash
cd /c/dev/datadatdat-remote-server/deploy/compose

# Authenticate to ECR
export ECR_REGISTRY=$(aws ecr describe-repositories --region us-west-2 --repository-names datadatdat/api-gateway --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Create .env file
cat > .env << EOF
REGISTRY=$ECR_REGISTRY/
VERSION=v1.1.0
EOF

# Pull and start
docker-compose pull
docker-compose up -d
```

**Option 3: Pull from Amazon ECR - Latest (Testing)**
```bash
cd /c/dev/datadatdat-remote-server/deploy/compose

# Authenticate to ECR
export ECR_REGISTRY=$(aws ecr describe-repositories --region us-west-2 --repository-names datadatdat/api-gateway --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Create .env file
cat > .env << EOF
REGISTRY=$ECR_REGISTRY/
VERSION=latest
EOF

# Pull and start
docker-compose pull
docker-compose up -d
```

#### Phase 7: Documentation Publication

##### 7.1 Release Documentation
```bash
# Documentation is automatically published when CLI is tagged
# The .github/workflows/docs-release.yml triggers on d3 CLI tags

# Manual verification:
# Check https://datadatdat.com for updated docs
# Verify version-specific docs are published
```

## Release Validation

### 1. Dependency Verification
After updating dependencies, verify compatibility:
```bash
cd /c/dev/datadatdat
go mod graph | grep datadatdat  # Check all internal dependencies
go list -m all | grep datadatdat  # Verify versions

# CRITICAL: Check for version mismatches in remote-sdk-go
go mod graph | grep datadatdat | grep remote-sdk-go
# Expected output: ALL 5 remote providers should use the SAME remote-sdk-go version
# Example of CORRECT output:
#   github.com/datadatdat/s3-remote-go@v1.1.0 github.com/datadatdat/remote-sdk-go@v1.1.0
#   github.com/datadatdat/ssh-remote-go@v1.1.0 github.com/datadatdat/remote-sdk-go@v1.1.0
#   github.com/datadatdat/s3web-remote-go@v1.1.0 github.com/datadatdat/remote-sdk-go@v1.1.0
#   github.com/datadatdat/nop-remote-go@v1.1.0 github.com/datadatdat/remote-sdk-go@v1.1.0
#   github.com/datadatdat/datadatdat-remote-go@v1.1.0 github.com/datadatdat/remote-sdk-go@v1.1.0

# If mismatches exist, STOP and update providers first before releasing CLI

# Verify no replace directives exist
grep "replace" go.mod && echo "❌ ERROR: Replace directives found!" || echo "✅ No replace directives"
```

### 1b. datadatdat-remote-server Dependency Verification
```bash
cd /c/dev/datadatdat-remote-server

# Verify no replace directives exist
grep "replace" go.mod && echo "❌ ERROR: Replace directives found!" || echo "✅ No replace directives"

# Check Go version compatibility
go version  # Should be Go 1.24+

# Verify dependencies are from GitHub (not local)
go list -m all | grep datadatdat
# Should show versions like v1.1.0, NOT local paths
```

## 🚨 CRITICAL: Dependency Conflict Resolution

**If you discover version conflicts after a release (like we did with v0.5.2), follow this emergency fix process:**

### Problem: Version Misalignment Detected
```bash
# Example of problematic output from: go mod graph | grep datadatdat | grep remote-sdk-go
# d3 github.com/datadatdat/remote-sdk-go@v0.2.8
# github.com/datadatdat/nop-remote-go@v0.2.4 github.com/datadatdat/remote-sdk-go@v0.2.6  # ❌ WRONG!
# github.com/datadatdat/s3-remote-go@v0.2.4 github.com/datadatdat/remote-sdk-go@v0.2.6   # ❌ WRONG!
```

### Emergency Fix Procedure
```bash
# 1. Update ALL remote providers to use the correct remote-sdk-go version
cd /c/dev/s3-remote-go && go get github.com/datadatdat/remote-sdk-go@v0.2.8 && go mod tidy
cd /c/dev/ssh-remote-go && go get github.com/datadatdat/remote-sdk-go@v0.2.8 && go mod tidy  
cd /c/dev/s3web-remote-go && go get github.com/datadatdat/remote-sdk-go@v0.2.8 && go mod tidy
cd /c/dev/nop-remote-go && go get github.com/datadatdat/remote-sdk-go@v0.2.8 && go mod tidy

# 2. Commit and release NEW versions of all providers
cd /c/dev/s3-remote-go && git add . && git commit -m "Update remote-sdk-go to v0.2.8" && git push
cd /c/dev/s3-remote-go && git tag v0.X.Y && git push origin v0.X.Y  # Increment patch version

# Repeat for all providers...

# 3. Update d3 CLI to use the NEW provider versions
cd /c/dev/datadatdat
go get github.com/datadatdat/s3-remote-go@v0.X.Y
go get github.com/datadatdat/ssh-remote-go@v0.X.Y
# ... update all providers
go mod tidy

# 4. Rebuild and release NEW d3 CLI patch version
export VERSION="v0.5.3"  # Increment patch version
make clean && make release
git add . && git commit -m "Fix dependency alignment" && git push
git tag $VERSION && git push origin $VERSION
gh release create $VERSION --title "$VERSION - Dependency Alignment Release" --notes "Critical patch to fix version conflicts" [artifacts...]

# 5. VERIFY the fix worked
go mod graph | grep datadatdat | grep remote-sdk-go
# Should now show ALL providers using the SAME remote-sdk-go version ✅
```

### 2. End-to-End Testing
```bash
cd /c/dev/datadatdat

# Critical: Run full e2e test suite
make e2e

# If tests fail due to corrupted state:
./d3.exe uninstall -f
make e2e

# CRITICAL FOR v1.1.0: Test datadatdat-remote-server integration
# This ensures the new remote server platform works correctly
make test-datadatdat-workflow

# Expected output: 
# ✅ All 20 tests passing (100%)
# Tests cover: health check, repo creation, commit, remote add, push, pull, checkout, cleanup

# If any test fails, DO NOT proceed with release
# Debug the issue and fix before continuing
```

**Test Breakdown:**
- Health check (1 test)
- Repository management (1 test)  
- Local operations (2 tests)
- Remote configuration (2 tests)
- Push operations (2 tests)
- Pull operations (2 tests)
- Checkout operations (1 test)
- Multi-commit workflow (3 tests)
- Cleanup operations (2 tests)
- Error handling (4 tests)

### 3. Docker Image Verification
```bash
# Verify new datadatdat-server image is published
docker pull datadatdat/datadatdat:latest
docker inspect datadatdat/datadatdat:latest

# Test with new CLI
./d3.exe install
./d3.exe status
```

## Automation Status (v1.3.0+)

### ✅ Fully Automated Components
- ✅ **remote-sdk-go**: Tag push → build → test → release → trigger cascade
- ✅ **Go providers (5)**: Tag push → build → test → publish release (no draft)
- ✅ **datadatdat-server**: Tag push → test → Docker publish to DockerHub
- ✅ **datadatdat-remote-server**: Tag push → build → E2E test → Docker publish to Amazon ECR
- ✅ **Dependency cascade**: SDK release → auto-create PRs in all providers

### 🎯 Semi-Automated (Human Review Required)
- 🔄 **Provider PR merges**: Auto-created PRs need manual review and merge
- 🔄 **Kotlin providers**: Tag push → Maven publish (automated), but need manual tagging
- 🔄 **datadatdat CLI**: Tag push → build → test → release (all automated), but critical testing required

### 🚀 Key Automation Achievements (v1.3.0)
1. **Unified Release Workflow**: remote-sdk-go release.yml does everything in one run
   - Builds and releases SDK
   - Updates go.mod in all 5 providers
   - Runs tests in each provider
   - Creates PRs automatically
   
2. **Immediate Publishing**: All releases publish immediately (no draft step)
   - Remote providers: published on tag push
   - CLI: published after tests pass
   - Servers: Docker images published after validation

3. **Smart Git Authentication**: Properly configured for Go private repos
   - Uses GO_MODULES_TOKEN (PAT) for private repo access
   - Correct URL format: `https://$TOKEN@github.com`
   - Global git config for Go module fetching

### 📋 What Still Requires Manual Steps
1. **PR Review**: 5 provider PRs need human review before merge (quality gate)
2. **Provider Tagging**: After merging PRs, manually tag each provider
3. **Kotlin Maven Publishing**: Manual git tagging triggers automation
4. **CLI Testing**: Manual E2E testing before accepting release

## Troubleshooting Common Issues

### Issue: Replace directives still present after cleanup
**Symptom**: Build fails with "module declares its path as ... but was required as ..."
**Solution**:
```bash
# Find all go.mod files with replace directives
cd /c/dev
find . -name "go.mod" -exec grep -l "replace" {} \;

# For each file found, manually remove replace directives
# Then run: go mod tidy
```

### Issue: datadatdat-remote-server E2E tests failing
**Symptom**: `make test-datadatdat-workflow` shows failures
**Common Causes:**
1. **datadatdat-remote-server not running**
   ```bash
   cd /c/dev/datadatdat-remote-server
   docker-compose -f deploy/compose/docker-compose.yml ps
   # All services should show "healthy" status
   ```

2. **Version mismatch between components**
   ```bash
   # Check d3 CLI is using correct provider versions
   cd /c/dev/datadatdat
   go list -m github.com/datadatdat/datadatdat-remote-go
   # Should show v1.1.0, not v1.0.0 or local path
   ```

3. **Port conflicts**
   ```bash
   # Check if ports 8080, 9000, 5432 are available
   netstat -an | grep :8080
   netstat -an | grep :9000
   netstat -an | grep :5432
   ```

4. **Old Docker images cached**
   ```bash
   # Clear old images and restart
   cd /c/dev/datadatdat-remote-server
   docker-compose -f deploy/compose/docker-compose.yml down -v
   docker-compose -f deploy/compose/docker-compose.yml pull
   docker-compose -f deploy/compose/docker-compose.yml up -d
   ```

### Issue: Failed e2e tests after dependency updates
**Solution**:
```bash
./d3.exe uninstall -f  # Critical cleanup step
make e2e  # Retry tests
```

### Issue: Version conflicts in go.mod  
**Example**: d3 depends on remote-sdk-go v1.1.0 but providers still use v1.0.0
**Solution**:
```bash
# Check for version mismatches
go mod graph | grep datadatdat | grep remote-sdk-go

# If mismatches found, update providers first before releasing CLI
cd /c/dev/s3-remote-go
go get github.com/datadatdat/remote-sdk-go@v1.1.0
go mod tidy
git add go.mod go.sum
git commit -m "Update remote-sdk-go to v1.1.0"
git push

# Repeat for ALL 5 providers:
# - s3-remote-go
# - ssh-remote-go  
# - s3web-remote-go
# - nop-remote-go
# - datadatdat-remote-go

# Then release ALL providers before CLI
# Finally update CLI dependencies
cd /c/dev/datadatdat
go get github.com/datadatdat/s3-remote-go@v1.1.0
# ... (repeat for all providers)
go mod download
go mod tidy
go clean -modcache  # If persistent issues
```

### Issue: Docker container won't start after datadatdat-server release
**Solution**:
```bash
# Check ZFS pools are properly set up
cd cleanslate
.\setup-zfs-pools.ps1 -Clean -VerifyDocker

# Restart Docker and retry
./d3.exe uninstall -f
./d3.exe install
```

### Issue: Missing VERSION variable in make release
**Current Gap**: The Makefile expects VERSION to be set externally
**Solution**: Always export VERSION before running make release:
```bash
export VERSION="v0.5.2"
make release
```

## Manual Release Checklist

### Pre-Release (1-2 days before)
- [ ] Plan version increments for all components
- [ ] Review documentation for accuracy
- [ ] Ensure OpenAPI spec reflects all server changes
- [ ] Coordinate release timing with team
- [ ] Prepare release notes and changelogs

### Foundation Release (Day 1 - Morning)
- [ ] Release remote-sdk-go with proper version tag
- [ ] Update and release all Go remote providers (parallel)
  - [ ] s3-remote-go
  - [ ] ssh-remote-go
  - [ ] s3web-remote-go
  - [ ] nop-remote-go
- [ ] Update and release all Kotlin remote providers (parallel)
  - [ ] s3-remote (publishes to Maven)
  - [ ] ssh-remote (publishes to Maven)
  - [ ] s3web-remote (publishes to Maven)
  - [ ] nop-remote (publishes to Maven)
- [ ] Verify all providers reference same remote-sdk-go version

### Client and CLI Release (Day 1 - Afternoon)
- [ ] Release datadatdat-client-go (regenerate from OpenAPI if needed)
- [ ] Update d3 CLI dependencies to latest versions
- [ ] Verify dependency compatibility with `go mod graph`
- [ ] Run full end-to-end test suite
- [ ] Build cross-platform CLI releases
- [ ] Create CLI git tag and GitHub release
- [ ] Upload CLI artifacts to GitHub release

### Container and Documentation (Day 1 - Evening)
- [ ] Release datadatdat-server (triggers Docker publishing automatically)
- [ ] Verify Docker images published to DockerHub
- [ ] Verify documentation published to datadatdat.com
- [ ] Test complete installation flow with new versions

### Post-Release Validation (Day 2)
- [ ] Integration testing across all platforms
- [ ] Download and installation verification
- [ ] Documentation accessibility verification  
- [ ] Community communication and announcements
- [ ] Update main README.md with new version numbers

### Emergency Procedures (If needed)
- [ ] Rollback procedures documented and tested
- [ ] Communication plan for critical issues
- [ ] Patch release process for urgent fixes

### Post-Release Validation (Same Day)

#### 1. Integration Testing
```bash
# Test the complete release pipeline
cd /c/dev/datadatdat

# Download and test new CLI
# wget/curl the new release from GitHub releases
# Test with fresh datadatdat-server container

./d3.exe install
./d3.exe run --name test-release -e POSTGRES_PASSWORD=password postgres
./d3.exe commit -m "Release validation test" test-release
./d3.exe log test-release
./d3.exe stop test-release
./d3.exe rm test-release
```

#### 2. Documentation Verification
```bash
# Verify documentation is live
curl -I https://datadatdat.com/  # Should return 200
# Check version-specific documentation exists
# Test getting started guide with new release
```

#### 3. Download Verification
```bash
# Test all platform downloads from GitHub releases
# Verify checksums if provided
# Test installation on clean systems
```

#### 4. Community Communication
```bash
# Update main README.md with new version numbers
# Announce in community channels (Discord, Slack, etc.)
# Post on community forums/social media
# Update any partner integrations
```

### Emergency Rollback Procedures

#### If CLI Release Fails
```bash
# Delete problematic GitHub release
# Revert git tag if needed
git tag -d v0.5.2
git push origin --delete v0.5.2

# Fix issues and re-release with patch version
```

#### If Docker Container Fails
```bash
# Docker images cannot be "untagged" once published to DockerHub
# Release a patch version immediately with fixes
# Communicate the issue to users immediately

# Emergency rollback for users:
docker pull datadatdat/datadatdat:v0.8.19  # Previous working version
# Update documentation with temporary workaround
```

#### If Maven Dependencies Fail
```bash
# Maven artifacts in S3 bucket cannot be deleted easily
# Release patch versions of affected components
# Ensure new versions resolve the issues
```

### Automation Gaps to Address

#### Critical Missing Automation
1. **d3 CLI release workflow** - No GitHub Action exists
2. **Cross-component dependency updates** - Manual coordination required
3. **Release validation testing** - No automated post-release verification
4. **Rollback automation** - No automated rollback procedures

#### Proposed GitHub Actions Improvements
```yaml
# .github/workflows/release.yml for d3 CLI
name: Release
on:
  create:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Build cross-platform
        run: make release VERSION=${GITHUB_REF#refs/tags/}
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          draft: true
          files: release/*
```
## Docker Container Analysis (Post-Rename Verification)

### Container Build Status Across Repositories

#### Automated Docker Publishing (GitHub Actions)
- ✅ **datadatdat-server**: Fully automated via GitHub Actions `.github/workflows/release.yml`
  - Uses Gradle build system with `docker.gradle.kts`
  - Dockerfile: `./server/docker/server.Dockerfile`
  - Publishes to DockerHub: `datadatdat/datadatdat:version` and `datadatdat/datadatdat:latest`
  - Multi-arch builds: `linux/amd64,linux/arm64`
  - Triggered by git tag push

- ✅ **localstack**: Has GitHub Actions (`.github/workflows/draft-release.yml`)
  - Manual Docker build process

#### Manual Docker Builds (No Automation)
- ��� **zfs-builder**: Has Dockerfile, no GitHub Actions
- ��� **zfs-linuxkit**: Has Dockerfile, no GitHub Actions  
- ��� **ssh-test-server**: Has Dockerfile, no GitHub Actions
- ��� **dynamodb-local**: Has Dockerfile, no GitHub Actions
- ��� **datadatdat** (CLI): Has Dockerfile for docs, uses GitHub Actions for docs only

#### No Docker Components
- ❌ **datadatdat-docker-proxy**: Name suggests Docker but no Dockerfile found
- ❌ **zfs-releases**: Has Dockerfile but unclear automation status

### Docker Build System Details

#### datadatdat-server (Primary Container)
**Build Method**: Gradle-based Docker builds
```bash
# Local build
./gradlew buildDockerServer

# Multi-arch publish (used by GitHub Actions)
./gradlew publishDockerServer -PserverImageName=datadatdat/datadatdat -PdatadatdatVersion=v0.8.20
```

**GitHub Actions Workflow**:
1. Tag creation triggers `.github/workflows/release.yml`
2. Runs full test suite including E2E Docker tests
3. Builds multi-architecture Docker image
4. Publishes to DockerHub with version and latest tags
5. Creates GitHub draft release

**Docker Registry**: DockerHub `datadatdat/datadatdat`

### Automation Gaps Identified
1. **Infrastructure containers** (zfs-builder, ssh-test-server, etc.) lack automated publishing
2. **datadatdat-docker-proxy** misleading name - no Docker functionality found
3. **Local development containers** require manual build and management

### Recommendations
1. **High Priority**: Verify datadatdat-server automation works post-rename
2. **Medium Priority**: Add automation for infrastructure containers if they're actively used
3. **Low Priority**: Consider renaming datadatdat-docker-proxy to clarify its purpose



## v1.0.0 Release Script

### Automated Release Execution

The complete v1.0.0 release process is automated via the **`release.sh`** script in the root of the datadatdat repository.

#### Usage
```bash
# Execute complete release process (recommended)
./release.sh

# Or run individual phases
./release.sh verify         # Check current version status
./release.sh foundation     # Release foundation components
./release.sh providers      # Release remote providers  
./release.sh infrastructure # Release plugin infrastructure
./release.sh core           # Release client and CLI
./release.sh docker         # Release server and Docker components
```

#### Script Features
- ✅ **Dependency-ordered execution** following proper release sequence
- ✅ **Version format validation** (Maven: `1.0.0`, Git tags: `v1.0.0`)
- ✅ **Color-coded output** with timestamps and status indicators
- ✅ **Error handling** with immediate exit on failures
- ✅ **Comprehensive verification** of post-release status
- ✅ **Modular execution** for partial releases or troubleshooting

#### Critical Version Requirements
The script automatically handles the critical version formatting differences:

- **Kotlin/Maven repositories**: Use `1.0.0` format (NO 'v' prefix)
  - `./gradlew publish -Pversion=1.0.0`
  - Affected: remote-sdk, command-executor, plugin-launcher, all Kotlin remotes

- **Git tags (all repositories)**: Use `v1.0.0` format (WITH 'v' prefix)
  - `git tag v1.0.0`
  - Applied to all 17+ repositories

- **Docker container**: Automatically publishes `datadatdat/datadatdat:1.0.0`
  - Triggered by datadatdat-server Git tag via GitHub Actions

#### Execution Order
1. **Foundation**: command-executor → remote-sdk → remote-sdk-go
2. **Providers**: Kotlin remotes (parallel) → Go remotes (parallel)
3. **Infrastructure**: plugin-launcher
4. **Core**: datadatdat-client-go → datadatdat CLI
5. **Docker**: datadatdat-docker-proxy → datadatdat-server

#### Pre-Release Checklist
```bash
# Verify all dependencies are at v1.0.0/1.0.0
./release.sh verify

# Check for uncommitted changes
git status

# Ensure you have push permissions to all repositories
```

#### Post-Release Verification
The script automatically verifies:
- ✅ All Git tags created successfully
- ✅ Docker image published to DockerHub
- ✅ CLI version commands functional
- ✅ Release process completion status

#### Emergency Procedures
If the release process fails at any stage:
1. **Check the error output** - script provides detailed logging
2. **Resume from specific phase** using individual commands
3. **Verify GitHub Actions** for datadatdat-server Docker publishing
4. **Manual verification** using the verification commands in the script


## Version System Fix (v1.0.0 Release)

### Issue Resolved: Hardcoded CLI Version ✅

**Previous Problem**: CLI version was hardcoded to "0.7.1" in `internal/app/commands/root.go`
- `d3 --version` always showed "d3 version 0.7.1" regardless of release tag
- VERSION environment variable was ignored during builds

**Solution Implemented**:

#### 1. Dynamic Version Variable
```go
// internal/app/commands/root.go
var Version = "dev"  // Default for development

func init() {
    rootCmd.Version = Version  // Use dynamic version
}
```

#### 2. Build-Time Version Injection
```makefile
# Makefile
VERSION ?= dev
LDFLAGS := -ldflags "-X datadatdat/internal/app/commands.Version=$(VERSION)"

build:
    go build $(LDFLAGS) -o $(TARGET) $(SOURCE)
```

#### 3. Release Process Integration
The `release.sh` script now:
- Sets `VERSION=1.1.0` environment variable
- Uses `make release` with proper version injection
- Generates correctly versioned release artifacts
- Ensures CLI reports correct version: `d3 version 1.1.0`

#### 4. Usage Examples
```bash
# Development build (shows "dev")
make build

# Versioned build
export VERSION="1.1.0"
make build
./build/d3 --version  # Shows "d3 version 1.1.0"

# Release build (automated by release.sh)
./release.sh  # Builds with v1.1.0 across all platforms
```

### Impact
- ✅ CLI version now matches release tags
- ✅ VERSION environment variable respected
- ✅ Release artifacts correctly named with version
- ✅ User support improved with accurate version reporting
- ✅ Automated via release script

---

## 🚀 Quick Reference: v1.1.0 Release Commands

### Complete Release Script (Copy-Paste Ready)

```bash
# Set version variables
export NEW_SDK_VERSION="v1.1.0"
export NEW_PROVIDER_VERSION="v1.1.0"
export NEW_CLIENT_VERSION="v1.1.0"
export VERSION="v1.1.0"
export KOTLIN_VERSION="1.1.0"  # No 'v' prefix for Maven

# ========================================
# PHASE 0: Clean Up Local Development State
# ========================================

# Remove replace directives from all go.mod files
cd /c/dev/datadatdat && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/datadatdat-remote-go && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/datadatdat-remote-server && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/nop-remote-go && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/s3-remote-go && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/s3web-remote-go && sed -i '/^replace/d' go.mod && go mod tidy
cd /c/dev/ssh-remote-go && sed -i '/^replace/d' go.mod && go mod tidy

# ========================================
# PHASE 1: Foundation - remote-sdk-go (FULLY AUTOMATED!)
# ========================================

cd /c/dev/remote-sdk-go
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION

# 🎉 ONE TAG TRIGGERS EVERYTHING:
# - Builds and tests SDK
# - Creates and publishes release
# - Updates go.mod in all 5 providers
# - Runs tests in each provider
# - Creates PRs in all 5 repos

# Monitor the workflow
gh run watch

# Wait ~2-3 minutes, then check for PRs
for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  echo "=== $repo ==="
  gh pr list --repo datadatdat/$repo --limit 1
done

# ========================================
# PHASE 2: Go Remote Providers - Review and Merge PRs
# ========================================

# Review each PR (tests already passed automatically)
# Then merge all 5 PRs
gh pr merge <PR_NUM> --repo datadatdat/s3-remote-go --squash --delete-branch
gh pr merge <PR_NUM> --repo datadatdat/ssh-remote-go --squash --delete-branch
gh pr merge <PR_NUM> --repo datadatdat/s3web-remote-go --squash --delete-branch
gh pr merge <PR_NUM> --repo datadatdat/nop-remote-go --squash --delete-branch
gh pr merge <PR_NUM> --repo datadatdat/datadatdat-remote-go --squash --delete-branch

# Tag each provider to trigger automatic release
for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  cd /c/dev/$repo
  git pull origin master
  git tag $NEW_PROVIDER_VERSION
  git push origin $NEW_PROVIDER_VERSION
done

# Releases publish automatically (no draft step)

# ========================================
# PHASE 3: Kotlin Remote Providers (All 5)
# ========================================

for provider in s3-remote ssh-remote s3web-remote nop-remote datadatdat-remote; do
  cd /c/dev/$provider
  ./gradlew build test
  git tag $KOTLIN_VERSION
  git push origin $KOTLIN_VERSION
  # GitHub Action automatically publishes to Maven
done

# ========================================
# PHASE 4: datadatdat-client-go (if needed)
# ========================================

cd /c/dev/datadatdat-client-go
git tag $NEW_CLIENT_VERSION
git push origin $NEW_CLIENT_VERSION

# ========================================
# PHASE 5: datadatdat CLI
# ========================================

cd /c/dev/datadatdat

# Update all dependencies
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go get github.com/datadatdat/s3-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/ssh-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/s3web-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/nop-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/datadatdat-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/datadatdat-client-go@$NEW_CLIENT_VERSION
go mod tidy

# Verify no conflicts
go mod graph | grep datadatdat | grep remote-sdk-go

# Test locally
make e2e
make test-datadatdat-workflow  # ALL 20 tests must pass

# CRITICAL: Build release binaries BEFORE committing
# This updates d3.exe and d3-linux in the root directory
make clean
VERSION=$VERSION make release
./d3.exe --version  # Verify: datadatdat version v1.1.0

# Commit everything INCLUDING the built binaries in root
git add go.mod go.sum internal/app/commands/root.go internal/app/providers/ Makefile RELEASE.md d3.exe d3-linux
git commit -m "Release $VERSION: Update all dependencies and fix issues"

# Push commits FIRST
git push origin master

# Then create and push tag (triggers GitHub Actions to create DRAFT release AND run E2E tests)
git tag $VERSION
git push origin $VERSION

# ⚠️ CRITICAL: Monitor E2E Test workflow (automatically triggered by tag push)
sleep 10
gh run watch  # Monitor until complete

# ✅ If tests PASS: Publish the draft release
gh release edit $VERSION --draft=false --latest

# ❌ If tests FAIL: Delete tag and fix issues
# git tag -d $VERSION && git push origin --delete $VERSION

# ========================================
# PHASE 5.5: Upload CLI Binaries to MinIO (for Web Downloads)
# ========================================

# REQUIRED: After publishing the GitHub release, upload binaries to MinIO
# This makes the CLI available for download via the web UI at /download

cd /c/dev/datadatdat-remote-server/scripts

# Run the upload script with the version
./upload-release-to-minio.sh --version $VERSION

# ⚠️ CRITICAL: Update download API tests to match new version
# The datadatdat-remote-server E2E tests check for specific version numbers
# These must be updated BEFORE releasing datadatdat-remote-server
cd /c/dev/datadatdat
# Edit tests/endtoend/remotes/datadatdat/datadatdat-workflow.bats
# Find and replace all occurrences of old version (e.g., v1.5.0) with $VERSION
# Tests to update:
#   - "download API: list versions returns v1.X.X"
#   - "download API: version details endpoint returns v1.X.X"  
#   - "download API: v1.X.X has linux-amd64 platform"
#   - "download API: v1.X.X has darwin-arm64 platform"
#   - "download API: v1.X.X has windows platform"
#   - "download API: platform metadata includes filename and size"
#   - "download API: binary download returns file for linux-amd64"
#   - "download API: binary download has correct content-type header"
#   - "download API: binary download has content-disposition header"
git add tests/endtoend/remotes/datadatdat/datadatdat-workflow.bats
git commit -m "Update download API tests to expect $VERSION"
git push origin master

# What this script does:
# 1. Downloads all release artifacts from GitHub
# 2. Extracts binaries from archives
# 3. Generates SHA256 checksums for each platform
# 4. Creates metadata.json with platform information
# 5. Uploads everything to MinIO bucket: datadatdat-releases
# 6. Organizes as: /$VERSION/{platform}/{binary + checksum}

# Verify upload succeeded
mc ls minio/datadatdat-releases/$VERSION/
# Should show:
#   metadata.json
#   darwin-amd64/
#   darwin-arm64/
#   linux-amd64/
#   linux-arm64/
#   windows/

# Test the download page
echo "Visit: http://localhost:3000/download"
echo "Should show $VERSION with all 5 platforms available"

# If upload fails, troubleshoot:
# - Check MinIO is running: docker ps | grep minio
# - Check mc is configured: mc alias list
# - Check GitHub release exists: gh release view $VERSION
# - Re-run with --force to overwrite: ./upload-release-to-minio.sh $VERSION --force

# Upload CLI binaries to MinIO (for web downloads)
cd /c/dev/datadatdat-remote-server/scripts
./upload-release-to-minio.sh $VERSION
mc ls minio/datadatdat-releases/$VERSION/  # Verify

# ========================================
# PHASE 6: datadatdat-server
# ========================================

cd /c/dev/datadatdat-server
git tag $VERSION
git push origin $VERSION
# GitHub Action automatically publishes Docker image

# ========================================
# PHASE 7: datadatdat-remote-server
# ========================================

cd /c/dev/datadatdat-remote-server

# Update dependencies
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go mod tidy

# Test locally
docker-compose -f deploy/compose/docker-compose.yml up -d
sleep 30
make test
cd /c/dev/datadatdat && make test-datadatdat-workflow
cd /c/dev/datadatdat-remote-server
docker-compose -f deploy/compose/docker-compose.yml down

# Release
git add go.mod go.sum
git commit -m "Update dependencies for v1.1.0 release"
git push origin master
git tag $VERSION
git push origin $VERSION
# GitHub Action automatically publishes 6 Docker images

# ========================================
# PHASE 8: Post-Release Validation
# ========================================

# Authenticate to Amazon ECR
export ECR_REGISTRY=$(aws ecr describe-repositories --region us-west-2 --repository-names datadatdat/api-gateway --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin $ECR_REGISTRY

# Verify Docker images
docker pull datadatdat/datadatdat:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-gateway:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-repo-manifest:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-ingest:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/api-download:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/worker:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/auth-server:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/web:v1.1.0
docker pull $ECR_REGISTRY/datadatdat/datadatdat-repo-web:v1.1.0

# Test with released images
cd /c/dev/datadatdat-remote-server
docker-compose -f deploy/compose/docker-compose.yml pull
docker-compose -f deploy/compose/docker-compose.yml up -d
sleep 30
cd /c/dev/datadatdat && make test-datadatdat-workflow
cd /c/dev/datadatdat-remote-server
docker-compose -f deploy/compose/docker-compose.yml down

echo "✅ v1.1.0 Release Complete!"
```

### Key Validation Commands

```bash
# Check for replace directives (should be empty)
grep -r "^replace" /c/dev/*/go.mod

# Verify dependency alignment
cd /c/dev/datadatdat
go mod graph | grep datadatdat | grep remote-sdk-go

# Run E2E tests
cd /c/dev/datadatdat
make test-datadatdat-workflow

# Check CLI version
./d3 --version  # Should show: d3 version 1.1.0
```

### Rollback Procedure

If issues are discovered after release:

```bash
# Delete problematic tags
git tag -d v1.1.0
git push origin --delete v1.1.0

# Delete GitHub releases (via web UI or gh CLI)
gh release delete v1.1.0 --yes

# Fix issues, then re-release with patch version v1.1.1
```

