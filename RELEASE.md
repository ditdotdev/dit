# Titan Ecosystem Release Process

This document outlines the comprehensive release process for the Titan data management platform. The ecosystem consists of multiple interdependent components that must be released in a specific order to maintain compatibility.

## 🚨 CRITICAL RELEASE CHECKLIST

**Before starting any release, review this checklist:**

- [ ] **Phase 1.1**: Release `remote-sdk-go` with new version (e.g., v0.2.8)
- [ ] **Phase 1.2**: ⚠️ **CRITICAL** - Update ALL 4 Go remote providers to use the SAME `remote-sdk-go` version
- [ ] **Phase 1.2**: Release NEW versions of all 4 remote providers
- [ ] **Phase 2**: Release Kotlin remote providers (if needed)
- [ ] **Phase 3**: Release `titan-client-go` (if needed)
- [ ] **Phase 4**: Update titan CLI dependencies to use NEW remote provider versions
- [ ] **Phase 4**: Verify dependency alignment: `go mod graph | grep datadatdat | grep remote-sdk-go`
- [ ] **Phase 4**: Release titan CLI with aligned dependencies
- [ ] **Phase 5**: Release titan-server (if needed)
- [ ] **Post-Release**: Validate entire ecosystem has consistent dependency versions

**⚠️ Phase 1.2 was previously missed and caused critical version conflicts requiring emergency patch releases!**

## Release Dependencies and Order

### Component Dependency Graph
```
remote-sdk-go (foundation)
    ↓
[s3-remote-go, ssh-remote-go, s3web-remote-go, nop-remote-go] (remote providers)
    ↓
titan-client-go (auto-generated from titan-server OpenAPI spec)
    ↓
titan (CLI - depends on all remote providers and client)
    ↓
titan-server (Docker container with ZFS + PostgreSQL)
```

### Release Order (Critical)
1. **remote-sdk-go** - Foundation SDK for all remote providers
2. **Remote providers** (can be done in parallel):
   - s3-remote-go
   - ssh-remote-go  
   - s3web-remote-go
   - nop-remote-go
3. **titan-client-go** - Auto-generated Go client
4. **titan** - Main CLI (depends on all above)
5. **titan-server** - Docker container (publishes to DockerHub)

### Supporting Components (Independent)
- **plugin-launcher** - Can be released independently
- **vexrun** - Testing framework, independent releases
- **zfs-builder**, **zfs-releases** - ZFS infrastructure, independent
- **Kotlin remotes** (s3-remote, ssh-remote, etc.) - JVM implementations, independent

## Version Strategy

### Current Versioning Scheme
- **titan**: v0.5.x (main CLI)
- **titan-server**: v0.8.x (Docker container)
- **titan-client-go**: v0.1.x (auto-generated client)
- **remote-sdk-go**: v0.2.x (foundation SDK)
- **Remote providers**: v0.2.x (aligned with SDK)

### Versioning Rules
1. **Semantic Versioning**: All components use semver (vMAJOR.MINOR.PATCH)
2. **Dependency Alignment**: Remote providers should align with remote-sdk-go versions
3. **CLI Independence**: Titan CLI version advances independently but must reference compatible dependency versions
4. **Server Alignment**: titan-server version should generally align with titan CLI for major releases

## Complete Titan Release Process - Step by Step

### Pre-Release Phase (1-2 days before)

#### 1. Pre-Release Planning
```bash
# Determine version increments for all components
# Check for breaking changes that require major version bumps
# Coordinate with team on release timing
```

#### 2. Documentation Review
```bash
cd /c/dev/titan-data.github.io
# Review and update documentation for new features
# Prepare release notes and changelog entries
```

#### 3. OpenAPI Specification Sync
```bash
cd /c/dev/titan-server
# Ensure OpenAPI spec (openapi/titan.yml) reflects all server changes
# This will trigger titan-client-go regeneration in next phase
```

### Release Phase Day

#### Phase 1: Foundation Components (Go Modules)

##### 1.1 Release remote-sdk-go
```bash
cd /c/dev/remote-sdk-go

# Ensure all tests pass
go test -v ./...

# Update version and create tag
export NEW_SDK_VERSION="v0.2.8"  # Increment appropriately - THIS VERSION WILL BE USED BY ALL PROVIDERS
git tag $NEW_SDK_VERSION
git push origin $NEW_SDK_VERSION

# GitHub Action automatically creates draft release
# Manually publish the draft release with release notes
```

**⚠️ IMPORTANT:** Note the `$NEW_SDK_VERSION` - this SAME version will be used by ALL remote providers in Step 1.2!

##### 1.2 Update and Release Remote Providers (Go) - CRITICAL STEP - DO NOT SKIP
**⚠️ WARNING: This step is MANDATORY and was previously missed, causing version conflicts**

For each provider (s3-remote-go, ssh-remote-go, s3web-remote-go, nop-remote-go):

```bash
cd /c/dev/s3-remote-go  # Repeat for each provider

# Update dependency to new remote-sdk-go version
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go mod tidy

# Run tests to ensure compatibility
go test -v ./...

# Create release
export NEW_PROVIDER_VERSION="v0.2.4"  # Increment appropriately
git add go.mod go.sum
git commit -m "Update remote-sdk-go to $NEW_SDK_VERSION"
git tag $NEW_PROVIDER_VERSION
git push origin master
git push origin $NEW_PROVIDER_VERSION

# GitHub Action automatically creates draft release
# Manually publish the draft release
```

**✅ VALIDATION: After completing all providers, verify version alignment:**
```bash
# Check that all providers are released and use the same remote-sdk-go version
cd /c/dev/titan
go get github.com/datadatdat/s3-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/ssh-remote-go@$NEW_PROVIDER_VERSION  
go get github.com/datadatdat/s3web-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/nop-remote-go@$NEW_PROVIDER_VERSION
go mod tidy
go mod graph | grep datadatdat | grep remote-sdk-go
# ALL providers should show the SAME remote-sdk-go version
```

#### Phase 2: Kotlin Remote Providers (Maven JARs) - Parallel Process

For each Kotlin remote (s3-remote, ssh-remote, s3web-remote, nop-remote):

```bash
cd /c/dev/s3-remote  # Repeat for each Kotlin remote

# Update remote-sdk dependency if needed
# Edit build.gradle.kts to update version

# Test build locally
./gradlew build test

# Create git tag (triggers automated Maven publishing)
export NEW_VERSION="v0.2.3"  # Increment appropriately  
git tag $NEW_VERSION
git push origin $NEW_VERSION

# GitHub Action automatically:
# - Builds and tests the JAR
# - Publishes to S3 Maven bucket (datadatdat-maven)
# - Creates GitHub draft release
```

#### Phase 3: Auto-Generated Client

##### 3.1 Regenerate titan-client-go
```bash
cd /c/dev/titan-client-go

# If OpenAPI spec changed, regenerate client
# (This may be automated or require manual trigger)

# Create release
export NEW_CLIENT_VERSION="v0.1.4"  # Increment appropriately
git tag $NEW_CLIENT_VERSION
git push origin $NEW_CLIENT_VERSION

# Simple tag-based release (no artifacts to build)
```

#### Phase 4: Main CLI Release

##### 4.1 Update titan CLI Dependencies
```bash
cd /c/dev/titan

# Update all dependencies to latest released versions
go get github.com/datadatdat/nop-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/remote-sdk-go@$NEW_SDK_VERSION
go get github.com/datadatdat/s3-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/s3web-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/ssh-remote-go@$NEW_PROVIDER_VERSION
go get github.com/datadatdat/titan-client-go@$NEW_CLIENT_VERSION
go mod tidy

# Verify no version conflicts
go mod graph | grep datadatdat | grep remote-sdk-go
# All providers should use same remote-sdk-go version
```

##### 4.2 Test and Build CLI
```bash
# Run full test suite
make e2e
# If tests fail: ./titan.exe uninstall -f && make e2e

# Build cross-platform releases  
export VERSION="v0.5.2"  # Increment appropriately
make release

# Creates artifacts in release/ directory:
# - titan-cli-$VERSION-windows_amd64.zip
# - titan-cli-$VERSION-darwin_amd64.zip  
# - titan-cli-$VERSION-darwin_arm64.zip
# - titan-cli-$VERSION-linux_amd64.tar
# - titan-cli-$VERSION-linux_arm64.tar
```

##### 4.3 Release CLI
```bash
# Create git tag and push
git add go.mod go.sum
git commit -m "Update dependencies for release $VERSION"
git tag $VERSION
git push origin master
git push origin $VERSION

# Use GitHub CLI to create release and upload artifacts
gh release create $VERSION \
  --title "$VERSION - [Release Title]" \
  --notes "## Release Notes

### 🔧 Dependency Updates
- List any updated components and versions

### 🎯 What's New/Fixed  
- Describe changes and fixes

### 📦 Artifacts
All cross-platform binaries included." \
  release/darwin-amd64/titan-cli-$VERSION-darwin_amd64.zip \
  release/darwin-arm64/titan-cli-$VERSION-darwin_arm64.zip \
  release/linux-amd64/titan-cli-$VERSION-linux_amd64.tar \
  release/linux-arm64/titan-cli-$VERSION-linux_arm64.tar \
  release/windows/titan-cli-$VERSION-windows_amd64.zip

# Verify release was created successfully
gh release view $VERSION
```

**⚠️ CRITICAL POST-RELEASE VALIDATION:**
```bash
# After releasing CLI, verify ENTIRE ecosystem has aligned dependencies
go mod graph | grep datadatdat | grep remote-sdk-go
# Should show ALL components using the SAME remote-sdk-go version
# If you see version conflicts, you MUST create a patch release to fix alignment
```

#### Phase 5: Docker Container Release

##### 5.1 Release titan-server
```bash
cd /c/dev/titan-server

# Ensure compatibility with new titan CLI version
# Update any version references if needed

# Create tag - this triggers automated publishing
export NEW_SERVER_VERSION="v0.8.20"  # Increment appropriately
git tag $NEW_SERVER_VERSION
git push origin $NEW_SERVER_VERSION

# GitHub Action automatically:
# - Runs full test suite including E2E tests
# - Builds multi-arch Docker image (linux/amd64, linux/arm64)  
# - Publishes to DockerHub as datadatdat/titan:$NEW_SERVER_VERSION
# - Tags and publishes datadatdat/titan:latest
# - Creates GitHub draft release
```

#### Phase 6: Documentation Publication

##### 6.1 Release Documentation
```bash
# Documentation is automatically published when CLI is tagged
# The .github/workflows/docs-release.yml triggers on titan CLI tags

# Manual verification:
# Check https://titan-data.io for updated docs
# Verify version-specific docs are published
```

## Release Validation

### 1. Dependency Verification
After updating dependencies, verify compatibility:
```bash
cd /c/dev/titan
go mod graph | grep datadatdat  # Check all internal dependencies
go list -m all | grep datadatdat  # Verify versions

# Check for version mismatches in remote-sdk-go
go mod graph | grep datadatdat | grep remote-sdk-go
# All remote providers should use the same remote-sdk-go version
# If mismatches exist, update providers first before releasing CLI
```

## 🚨 CRITICAL: Dependency Conflict Resolution

**If you discover version conflicts after a release (like we did with v0.5.2), follow this emergency fix process:**

### Problem: Version Misalignment Detected
```bash
# Example of problematic output from: go mod graph | grep datadatdat | grep remote-sdk-go
# titan github.com/datadatdat/remote-sdk-go@v0.2.8
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

# 3. Update titan CLI to use the NEW provider versions
cd /c/dev/titan
go get github.com/datadatdat/s3-remote-go@v0.X.Y
go get github.com/datadatdat/ssh-remote-go@v0.X.Y
# ... update all providers
go mod tidy

# 4. Rebuild and release NEW titan CLI patch version
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
# Critical: Run full e2e test suite
make e2e

# If tests fail due to corrupted state:
./titan.exe uninstall -f
make e2e
```

### 3. Docker Image Verification
```bash
# Verify new titan-server image is published
docker pull datadatdat/titan:latest
docker inspect datadatdat/titan:latest

# Test with new CLI
./titan.exe install
./titan.exe status
```

## Automation Opportunities

### Current Automation Status
- ✅ **titan-server**: Fully automated via GitHub Actions on tag push
- ✅ **Remote providers (Go)**: Automated workflows exist, just need tag push + manual release
- ❌ **titan CLI**: No automated workflow - manual build and release upload required
- ❌ **Cross-component coordination**: No automation for dependency updates

### Proposed Automation Improvements

#### 1. Titan CLI Release Workflow
Create `.github/workflows/release.yml` in titan repo:
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

### Issue: Failed e2e tests after dependency updates
**Solution**:
```bash
./titan.exe uninstall -f  # Critical cleanup step
make e2e  # Retry tests
```

### Issue: Version conflicts in go.mod  
**Example**: titan depends on remote-sdk-go v0.2.5 but providers still use v0.2.4
**Solution**:
```bash
# Check for version mismatches
go mod graph | grep datadatdat | grep remote-sdk-go

# If mismatches found, update providers first before releasing CLI
cd /c/dev/s3-remote-go
go get github.com/datadatdat/remote-sdk-go@v0.2.5
go mod tidy
# Repeat for all providers, then release them before CLI

go mod download
go mod tidy
go clean -modcache  # If persistent issues
```

### Issue: Docker container won't start after titan-server release
**Solution**:
```bash
# Check ZFS pools are properly set up
cd cleanslate
.\setup-zfs-pools.ps1 -Clean -VerifyDocker

# Restart Docker and retry
./titan.exe uninstall -f
./titan.exe install
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
- [ ] Release titan-client-go (regenerate from OpenAPI if needed)
- [ ] Update titan CLI dependencies to latest versions
- [ ] Verify dependency compatibility with `go mod graph`
- [ ] Run full end-to-end test suite
- [ ] Build cross-platform CLI releases
- [ ] Create CLI git tag and GitHub release
- [ ] Upload CLI artifacts to GitHub release

### Container and Documentation (Day 1 - Evening)
- [ ] Release titan-server (triggers Docker publishing automatically)
- [ ] Verify Docker images published to DockerHub
- [ ] Verify documentation published to titan-data.io
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
cd /c/dev/titan

# Download and test new CLI
# wget/curl the new release from GitHub releases
# Test with fresh titan-server container

./titan.exe install
./titan.exe run --name test-release -e POSTGRES_PASSWORD=password postgres
./titan.exe commit -m "Release validation test" test-release
./titan.exe log test-release
./titan.exe stop test-release
./titan.exe rm test-release
```

#### 2. Documentation Verification
```bash
# Verify documentation is live
curl -I https://titan-data.io/  # Should return 200
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
docker pull datadatdat/titan:v0.8.19  # Previous working version
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
1. **titan CLI release workflow** - No GitHub Action exists
2. **Cross-component dependency updates** - Manual coordination required
3. **Release validation testing** - No automated post-release verification
4. **Rollback automation** - No automated rollback procedures

#### Proposed GitHub Actions Improvements
```yaml
# .github/workflows/release.yml for titan CLI
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