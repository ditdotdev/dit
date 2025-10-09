# Titan Ecosystem Release Process

This document outlines the comprehensive release process for the Titan data management platform. The ecosystem consists of multiple interdependent components that must be released in a specific order to maintain compatibility.

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

## Step-by-Step Release Process

### Phase 1: Prepare Foundation Components

#### 1. Release remote-sdk-go
```bash
cd /c/dev/remote-sdk-go

# Ensure all tests pass
go test -v ./...

# Update version and create tag
export NEW_VERSION="v0.2.6"  # Increment appropriately
git tag $NEW_VERSION
git push origin $NEW_VERSION

# Publish draft release on GitHub (manual step)
# Go to https://github.com/datadatdat/remote-sdk-go/releases
# Create release from tag with release notes
```

#### 2. Update Remote Providers (Parallel)
For each remote provider (s3-remote-go, ssh-remote-go, s3web-remote-go, nop-remote-go):

```bash
cd /c/dev/s3-remote-go  # Repeat for each provider

# Update dependency to new remote-sdk-go version
go get github.com/datadatdat/remote-sdk-go@v0.2.6
go mod tidy

# Run tests to ensure compatibility
go test -v ./...

# Create release
export NEW_VERSION="v0.2.4"  # Increment appropriately
git add go.mod go.sum
git commit -m "Update remote-sdk-go to $NEW_VERSION"
git tag $NEW_VERSION
git push origin master
git push origin $NEW_VERSION

# Publish draft release on GitHub (manual step)
```

### Phase 2: Update and Release titan-client-go

```bash
cd /c/dev/titan-client-go

# titan-client-go is auto-generated from titan-server OpenAPI spec
# Ensure it's up to date with latest server changes
# This may require coordination with titan-server team

# Create release (no dependency updates needed as it's auto-generated)
export NEW_VERSION="v0.1.4"  # Increment appropriately
git tag $NEW_VERSION
git push origin $NEW_VERSION

# Publish draft release on GitHub (manual step)
```

### Phase 3: Update and Release titan CLI

```bash
cd /c/dev/titan

# Update all dependencies to latest versions
go get github.com/datadatdat/nop-remote-go@v0.2.4
go get github.com/datadatdat/remote-sdk-go@v0.2.6
go get github.com/datadatdat/s3-remote-go@v0.2.4
go get github.com/datadatdat/s3web-remote-go@v0.2.3
go get github.com/datadatdat/ssh-remote-go@v0.2.3
go get github.com/datadatdat/titan-client-go@v0.1.4
go mod tidy

# Run full test suite
make e2e
# If tests fail, run: ./titan.exe uninstall -f
# Then retry: make e2e

# Build cross-platform releases
export VERSION="v0.5.2"  # Increment appropriately
make release

# Create git tag and push
git add go.mod go.sum
git commit -m "Update dependencies for release $VERSION"
git tag $VERSION
git push origin master
git push origin $VERSION

# Manual release creation required (no automated workflow exists)
# Upload release artifacts from release/ directory to GitHub
```

### Phase 4: Release titan-server (Docker Container)

```bash
cd /c/dev/titan-server

# Ensure compatibility with new titan CLI version
# Update any version references if needed

# Create tag - this triggers automated Docker publishing
export NEW_VERSION="v0.8.20"  # Increment appropriately
git tag $NEW_VERSION
git push origin $NEW_VERSION

# Automated workflow publishes:
# - Docker image to datadatdat/titan:latest and datadatdat/titan:$NEW_VERSION
# - Creates GitHub draft release
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

### Pre-Release
- [ ] All components have passing tests
- [ ] Dependencies are up to date and compatible
- [ ] Version numbers are properly incremented
- [ ] Release notes are prepared

### Release Execution
- [ ] Release remote-sdk-go
- [ ] Release all remote providers (verify dependency updates)
- [ ] Release titan-client-go
- [ ] Release titan CLI (includes cross-platform builds)
- [ ] Release titan-server (triggers Docker publishing)

### Post-Release Validation
- [ ] All GitHub releases are published
- [ ] Docker images are available on DockerHub
- [ ] End-to-end tests pass with new versions
- [ ] Documentation is updated with new version numbers

### Communication
- [ ] Update main README.md with latest version
- [ ] Announce release in community channels
- [ ] Update any deployment documentation