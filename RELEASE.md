cd /c/dev/datadatdat
export VERSION="v1.2.0"

# Upload all release artifacts to the draft release
gh release upload $VERSION \
  release/darwin-amd64/datadatdat-cli-$VERSION-darwin_amd64.zip \
  release/darwin-arm64/datadatdat-cli-$VERSION-darwin_arm64.zip \
  release/linux-amd64/datadatdat-cli-$VERSION-linux_amd64.tar \
  release/linux-arm64/datadatdat-cli-$VERSION-linux_arm64.tar \
  release/windows/datadatdat-cli-$VERSION-windows_amd64.zip

# Verify artifacts were uploaded
gh release view $VERSIONcd /c/dev/datadatdat
export VERSION="v1.2.0"

# Upload all release artifacts to the draft release
gh release upload $VERSION \
  release/darwin-amd64/datadatdat-cli-$VERSION-darwin_amd64.zip \
  release/darwin-arm64/datadatdat-cli-$VERSION-darwin_arm64.zip \
  release/linux-amd64/datadatdat-cli-$VERSION-linux_amd64.tar \
  release/linux-arm64/datadatdat-cli-$VERSION-linux_arm64.tar \
  release/windows/datadatdat-cli-$VERSION-windows_amd64.zip

# Verify artifacts were uploaded
gh release view $VERSION# Datadatdat Ecosystem Release Process

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
- **5 new Docker images**: Published to GHCR as ghcr.io/datadatdat/[service]:v1.1.0

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

### Phase 1: Foundation
- [ ] **Phase 1.1**: Release `remote-sdk-go` with new version (v1.1.0)
- [ ] **Phase 1.2**: ⚠️ **CRITICAL** - Update ALL 6 Go remote providers to use the SAME `remote-sdk-go` version
- [ ] **Phase 1.2**: Release NEW versions of all 6 remote providers (including datadatdat-remote-go)
- [ ] **Phase 1.3**: ⚠️ **REQUIRED** - Release `remote-sdk` (Kotlin/Maven) version 1.1.0 BEFORE Phase 2

### Phase 2: Kotlin Providers  
- [ ] **Phase 2**: Release 6 Kotlin remote providers with version 1.1.0 (NO 'v' prefix)
- [ ] **Phase 2**: Verify datadatdat-remote publishes BOTH client and server artifacts to Maven

### Phase 3-5: Core Components
- [ ] **Phase 3**: Release `datadatdat-client-go` v1.1.0 (if needed)
- [ ] **Phase 4**: Remove replace directives from datadatdat/go.mod
- [ ] **Phase 4**: Update datadatdat CLI dependencies to use NEW remote provider versions
- [ ] **Phase 4**: Verify dependency alignment: `go mod graph | grep datadatdat | grep remote-sdk-go`
- [ ] **Phase 4**: Run `make test-datadatdat-workflow` - ALL tests must pass
- [ ] **Phase 4**: Release datadatdat CLI with aligned dependencies
- [ ] **Phase 5**: Release datadatdat-server v1.1.0

### Phase 6: Remote Server Platform
- [ ] **Phase 6.1**: Remove replace directives from datadatdat-remote-server/go.mod
- [ ] **Phase 6.2**: Run local E2E tests - `make test-datadatdat-workflow` must pass
- [ ] **Phase 6.3**: Release datadatdat-remote-server v1.1.0 (6 Docker images)
- [ ] **Phase 6.4**: Verify all 6 Docker images published to DockerHub
- [ ] **Phase 6.5**: Run E2E tests against released images

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
- [ ] **Phase 4**: Release datadatdat CLI v1.1.0
- [ ] **Phase 5**: Release datadatdat-server v1.1.0
- [ ] **Phase 6**: Release datadatdat-remote-server v1.1.0 (6 Docker images)

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
datadatdat (CLI - depends on all remote providers and client)
    ↓
datadatdat-server (Docker container with ZFS + PostgreSQL)
    ↓
datadatdat-remote-server (Microservices platform - "GitHub for Data")
```

### Release Order (Critical)
1. **remote-sdk-go** - Foundation SDK for all remote providers
2. **Remote providers** (can be done in parallel):
   - s3-remote-go
   - ssh-remote-go  
   - s3web-remote-go
   - nop-remote-go
   - **datadatdat-remote-go** - NEW: Provider for datadatdat-remote-server
3. **Kotlin remote providers** (can be done in parallel):
   - s3-remote
   - ssh-remote
   - s3web-remote
   - nop-remote
   - **datadatdat-remote** - NEW: Server-side provider for datadatdat-remote-server
4. **datadatdat-client-go** - Auto-generated Go client
5. **datadatdat** - Main CLI (depends on all above)
6. **datadatdat-server** - Docker container (publishes to DockerHub)
7. **datadatdat-remote-server** - NEW: Microservices platform (publishes 6 Docker images)

### Supporting Components (Independent)
- **plugin-launcher** - Can be released independently
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

##### 1.1 Release remote-sdk-go
```bash
cd /c/dev/remote-sdk-go

# Build the binary (needed for tests on Windows)
go build -o build/echo.exe ./cmd/echo  # Windows
go build -o build/echo ./cmd/echo      # Linux/Mac

# Ensure all tests pass
go test ./...

# Update version and create tag
export NEW_SDK_VERSION="v1.1.0"  # Increment appropriately - THIS VERSION WILL BE USED BY ALL PROVIDERS
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION

# Wait for GitHub Action to complete successfully
gh run list --workflow=release.yml --limit 5
# Look for ✓ status on the v1.1.0 tag

# Verify draft release was created with binary attached
gh release list --limit 5
# Should show "v1.1.0  Draft"

gh release view $NEW_SDK_VERSION
# Verify the echo-linux binary is attached

# Publish the draft release (CRITICAL STEP!)
gh release edit $NEW_SDK_VERSION --draft=false --latest

# Verify it's published
gh release list --limit 5
# Should now show "v1.1.0  Latest"
```

**⚠️ CRITICAL STEPS:**
1. **Build & Test**: All tests must pass before tagging
2. **Tag & Push**: Creates the tag and triggers GitHub Actions
3. **Verify Workflow**: Wait for GitHub Actions to complete (✓ status)
4. **Verify Draft**: Check that draft release exists with binary
5. **Publish Release**: Use `gh release edit` to publish the draft
6. **Confirm**: Verify it shows as "Latest" not "Draft"

**⚠️ IMPORTANT:** Note the `$NEW_SDK_VERSION` - this SAME version will be used by ALL remote providers in Step 1.2!

##### 1.2 Update and Release Remote Providers (Go) - CRITICAL STEP - DO NOT SKIP
**⚠️ WARNING: This step is MANDATORY and was previously missed, causing version conflicts**

For each provider (s3-remote-go, ssh-remote-go, s3web-remote-go, nop-remote-go, datadatdat-remote-go):

```bash
cd /c/dev/s3-remote-go  # Repeat for each provider

# Update dependency to new remote-sdk-go version
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go mod tidy

# Run tests to ensure compatibility
go test ./...

# Commit and create release
export NEW_PROVIDER_VERSION="v1.1.0"  # Use v1.1.0 for this release
git add go.mod go.sum
git commit -m "Update remote-sdk-go to $NEW_SDK_VERSION"
git push origin master
git tag $NEW_PROVIDER_VERSION
git push origin $NEW_PROVIDER_VERSION

# Wait for GitHub Action to complete
gh run list --workflow=release.yml --limit 3
# Look for ✓ status

# Verify and publish draft release
gh release list --limit 3
gh release view $NEW_PROVIDER_VERSION
# Verify binary is attached

# Publish the draft release (CRITICAL!)
gh release edit $NEW_PROVIDER_VERSION --draft=false --latest

# Verify published
gh release list --limit 3
```

**✅ VALIDATION: After completing all providers, verify version alignment:**
```bash
# Check that all providers are released and use the same remote-sdk-go version
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

##### 4.2 Build Release Artifacts
```bash
# CRITICAL: Build FIRST - E2E tests need to run against this build
export VERSION="v1.2.0"  # Set your version
make clean  # Clean all caches
VERSION=$VERSION make release

# Creates artifacts in release/ directory:
# - datadatdat-cli-$VERSION-windows_amd64.zip
# - datadatdat-cli-$VERSION-darwin_amd64.zip  
# - datadatdat-cli-$VERSION-darwin_arm64.zip
# - datadatdat-cli-$VERSION-linux_amd64.tar
# - datadatdat-cli-$VERSION-linux_arm64.tar

# IMPORTANT: Also copies d3.exe and d3-linux to ROOT directory
# These root binaries should be committed as part of the release

# Verify version in binary
./d3.exe --version  # Should show: datadatdat version v1.2.0
```

##### 4.3 Test Locally
```bash
# Run full test suite including datadatdat-remote-server E2E tests
# These tests use the d3.exe built in step 4.2
make e2e
# If tests fail: ./d3.exe uninstall -f && make e2e

# CRITICAL: Test datadatdat remote workflow specifically
make test-datadatdat-workflow
# This runs the E2E tests for datadatdat-remote-server integration
# All 20 tests must pass before proceeding with release
```

##### 4.4 Commit, Tag, and Push
```bash
# Stage ALL changes including:
# - Dependency updates (go.mod, go.sum)
# - Code changes (if any)
# - Built binaries in ROOT (d3.exe, d3-linux) - CRITICAL!
# - Release artifacts stay in release/ directory (not committed)

git add go.mod go.sum internal/app/commands/root.go internal/app/providers/ Makefile RELEASE.md d3.exe d3-linux

# Commit with comprehensive message
git commit -m "Release $VERSION: Update all dependencies and fix issues

- Updated remote-sdk-go to v1.1.0
- Updated all Go remote providers to v1.1.0
- Added all remote provider imports
- Fixed bugs and removed debug output
- Built release binaries with $VERSION version
- All E2E tests passing"

# Push commits first
git push origin master

# Create tag and push
git tag $VERSION
git push origin $VERSION
```

##### 4.5 Create Draft Release with Artifacts
```bash
# NOTE: The datadatdat CLI repo does NOT have automated release creation
# We must manually create the draft release and upload artifacts

# Create draft release with all artifacts
gh release create $VERSION --draft \
  --title "$VERSION - Authorization Header Support" \
  --notes "## Release $VERSION

### 🔧 Key Changes
- [List your changes here]

### 🧪 Testing
- All E2E tests passing
- Full integration testing completed

### 📦 Artifacts
Cross-platform binaries included for all supported platforms." \
  release/darwin-amd64/datadatdat-cli-$VERSION-darwin_amd64.zip \
  release/darwin-arm64/datadatdat-cli-$VERSION-darwin_arm64.zip \
  release/linux-amd64/datadatdat-cli-$VERSION-linux_amd64.tar \
  release/linux-arm64/datadatdat-cli-$VERSION-linux_arm64.tar \
  release/windows/datadatdat-cli-$VERSION-windows_amd64.zip

# Verify draft release was created with all artifacts
gh release view $VERSION
```

##### 4.7 Run E2E Test Workflow (CRITICAL GATE)
**⚠️ DO NOT proceed to publishing until this step passes!**

```bash
# Manually trigger the "End to End Test" workflow on the tag
# Note: This workflow runs on workflow_dispatch (manual trigger) or nightly schedule
gh workflow run end-to-end-test.yml --ref $VERSION

# Wait a moment for workflow to start, then monitor it
sleep 10
gh run watch

# Alternative: Check workflow status
gh run list --workflow=end-to-end-test.yml --limit 5

# Or view in browser
gh workflow view end-to-end-test.yml --web

# CRITICAL DECISION POINT:
# ✅ If E2E workflow PASSES → Proceed to step 4.8 (Publish Release)
# ❌ If E2E workflow FAILS → DO NOT PUBLISH
#    - Delete the release and tag: gh release delete $VERSION --yes && git tag -d $VERSION && git push origin --delete $VERSION
#    - Fix the issues
#    - Restart from step 4.1 with a new version (e.g., v1.2.1)
```

**Why This Step is Critical:**
- Tests the EXACT code that will be released (the tag)
- Validates all dependencies work together in CI environment
- Catches issues before users download the release
- Prevents publishing broken releases

##### 4.8 Publish Release (Only After E2E Tests Pass)
```bash
# Now publish the release after E2E tests confirm it works

# Check that draft release exists with artifacts
gh release list --limit 5
gh release view $VERSION

# Publish the draft release
gh release edit $VERSION --draft=false --latest

# Verify release is now published
gh release view $VERSION

# Alternative: If you want to add release notes while publishing
gh release edit $VERSION \
  --draft=false \
  --latest \
  --notes "## Release v1.1.0

### 🔧 Dependency Updates
- remote-sdk-go v1.1.0
- All Go remote providers v1.1.0 (s3-remote-go v1.1.2)
- All Kotlin remote providers 1.1.0

### 🎯 What's New
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

#### Phase 5: Docker Container Release

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

##### 6.3 Release datadatdat-remote-server to GHCR
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
#   - Builds 5 Docker images (linux/amd64):
#     * ghcr.io/datadatdat/api-gateway:v1.1.0 and :latest
#     * ghcr.io/datadatdat/api-repo-manifest:v1.1.0 and :latest
#     * ghcr.io/datadatdat/api-ingest:v1.1.0 and :latest
#     * ghcr.io/datadatdat/api-download:v1.1.0 and :latest
#     * ghcr.io/datadatdat/worker:v1.1.0 and :latest
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
# Job 3: Publish to GHCR (only if tests pass)
#   - Authenticates to ghcr.io using GITHUB_TOKEN
#   - Pushes all version tags (v1.1.0)
#   - Pushes all latest tags
#   - Creates GitHub draft release with deployment instructions
```

##### 6.4 Authenticate to GHCR and Verify Release
```bash
# Authenticate to GitHub Container Registry
# You need a GitHub Personal Access Token with read:packages scope
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# OR: Use gh CLI to automatically authenticate
gh auth token | docker login ghcr.io -u $(gh api user -q .login) --password-stdin

# Pull all published images from GHCR (private registry)
docker pull ghcr.io/datadatdat/api-gateway:v1.1.0
docker pull ghcr.io/datadatdat/api-repo-manifest:v1.1.0
docker pull ghcr.io/datadatdat/api-ingest:v1.1.0
docker pull ghcr.io/datadatdat/api-download:v1.1.0
docker pull ghcr.io/datadatdat/worker:v1.1.0

# Verify latest tags are also available
docker pull ghcr.io/datadatdat/api-gateway:latest

# Check GitHub release
gh release view v1.1.0 --repo datadatdat/datadatdat-remote-server

# Verify all 5 images are listed in the release notes
```

##### 6.5 Deploy and Test with GHCR Images
```bash
# CRITICAL: Verify the released images work correctly

cd /c/dev/datadatdat-remote-server/deploy/compose

# Authenticate to GHCR (if not already done)
gh auth token | docker login ghcr.io -u $(gh api user -q .login) --password-stdin

# Create .env file to use GHCR and specific version
cat > .env << 'EOF'
REGISTRY=ghcr.io/
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

**Option 2: Pull from GHCR - Specific Version (Production)**
```bash
cd /c/dev/datadatdat-remote-server/deploy/compose

# Create .env file
cat > .env << 'EOF'
REGISTRY=ghcr.io/
VERSION=v1.1.0
EOF

# Authenticate to GHCR
gh auth token | docker login ghcr.io -u $(gh api user -q .login) --password-stdin

# Pull and start
docker-compose pull
docker-compose up -d
```

**Option 3: Pull from GHCR - Latest (Testing)**
```bash
cd /c/dev/datadatdat-remote-server/deploy/compose

# Create .env file
cat > .env << 'EOF'
REGISTRY=ghcr.io/
VERSION=latest
EOF

# Authenticate and pull
gh auth token | docker login ghcr.io -u $(gh api user -q .login) --password-stdin
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

## Automation Opportunities

### Current Automation Status
- ✅ **datadatdat-server**: Fully automated via GitHub Actions on tag push
- ✅ **Remote providers (Go)**: Automated workflows exist, just need tag push + manual release
- ❌ **d3 CLI**: No automated workflow - manual build and release upload required
- ❌ **Cross-component coordination**: No automation for dependency updates

### Proposed Automation Improvements

#### 1. Datadatdat CLI Release Workflow
Create `.github/workflows/release.yml` in d3 repo:
```yaml
name: Release
on:
  create:
    tags:
      - 'v*'
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

#### 2. Dependency Update Automation
- Use dependabot or similar to track internal dependency updates
- Create scripts to batch update all remote providers when SDK changes

#### 3. Release Coordination Script
Create a master script that:
1. Determines which components need releases based on changes
2. Calculates correct release order
3. Automates the entire release pipeline

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
# PHASE 1: Foundation - remote-sdk-go
# ========================================

cd /c/dev/remote-sdk-go
go test -v ./...
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION
# Wait for GitHub Action, then publish draft release

# ========================================
# PHASE 2: Go Remote Providers (All 5)
# ========================================

for provider in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go datadatdat-remote-go; do
  cd /c/dev/$provider
  go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
  go mod tidy
  go test -v ./...
  git add go.mod go.sum
  git commit -m "Update remote-sdk-go to $NEW_SDK_VERSION"
  git push origin master
  git tag $NEW_PROVIDER_VERSION
  git push origin $NEW_PROVIDER_VERSION
  # Wait for GitHub Action, then publish draft release
done

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

# Then create and push tag (triggers GitHub Actions to create DRAFT release)
git tag $VERSION
git push origin $VERSION

# ⚠️ CRITICAL: Run E2E Test workflow on the tag BEFORE publishing
gh workflow run end-to-end-test.yml --ref $VERSION
sleep 10
gh run watch  # Monitor until complete

# ✅ If tests PASS: Publish the draft release
gh release edit $VERSION --draft=false --latest

# ❌ If tests FAIL: Delete tag and fix issues
# git tag -d $VERSION && git push origin --delete $VERSION

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

# Verify Docker images
docker pull datadatdat/datadatdat:v1.1.0
docker pull datadatdat/api-gateway:v1.1.0
docker pull datadatdat/api-repo-manifest:v1.1.0
docker pull datadatdat/api-ingest:v1.1.0
docker pull datadatdat/api-download:v1.1.0
docker pull datadatdat/worker:v1.1.0
docker pull datadatdat/datadatdat-provider-http:v1.1.0

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

