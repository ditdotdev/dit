# Automation Plan for v1.3.0 Release

## Goals
1. Reduce manual steps from ~50 to ~10
2. Prevent version conflicts through automation
3. Enable one-command release pipeline
4. Improve release reliability and speed

## Phase 1: Dependency Update Automation (Highest ROI)

### 1.1 Auto-Update Go Providers When SDK Releases
**Location**: `remote-sdk-go/.github/workflows/dependency-cascade.yml`

```yaml
name: Cascade Dependency Updates
on:
  release:
    types: [published]  # Triggers when draft is published

jobs:
  update-providers:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        repo:
          - s3-remote-go
          - ssh-remote-go
          - s3web-remote-go
          - nop-remote-go
          - datadatdat-remote-go
    steps:
      - name: Update go.mod in ${{ matrix.repo }}
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          
          # Clone provider repo
          git clone https://${{ secrets.GO_MODULES_TOKEN }}@github.com/datadatdat/${{ matrix.repo }}.git
          cd ${{ matrix.repo }}
          
          # Update dependency
          go get github.com/datadatdat/remote-sdk-go@$VERSION
          go mod tidy
          go test ./...
          
          # Create PR
          git checkout -b auto-update-sdk-$VERSION
          git add go.mod go.sum
          git commit -m "Auto-update remote-sdk-go to $VERSION"
          git push origin auto-update-sdk-$VERSION
          
          gh pr create \
            --title "Update remote-sdk-go to $VERSION" \
            --body "Automated dependency update from remote-sdk-go release" \
            --base master
```

**Benefits:**
- Eliminates Step 1.2 (5 manual repo updates)
- Creates PRs for review instead of direct commits
- Runs tests before creating PR
- Saves ~30 minutes per release

### 1.2 Auto-Publish Draft Releases After Tests Pass
**Location**: Add to all provider `release.yml` workflows

```yaml
jobs:
  release:
    # ... existing build steps ...
    
  test:
    needs: release
    runs-on: ubuntu-latest
    steps:
      - name: Run tests
        run: go test ./...
  
  publish:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Publish draft release
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          gh release edit $VERSION --draft=false --latest
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Benefits:**
- Eliminates 10+ manual `gh release edit` commands
- Releases only after tests pass
- Faster release cycle

## Phase 2: CLI Release Automation

### 2.1 Datadatdat CLI Release Workflow
**Location**: `datadatdat/.github/workflows/release.yml`

```yaml
name: Release
on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25.1'
      
      - name: Configure Git for private modules
        run: |
          git config --global url."https://x-access-token:${{ secrets.GO_MODULES_TOKEN }}@github.com/".insteadOf "https://github.com/"
      
      - name: Build cross-platform releases
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          make clean
          VERSION=$VERSION make release
      
      - name: Create draft release
        uses: softprops/action-gh-release@v1
        with:
          draft: true
          files: |
            release/darwin-amd64/datadatdat-cli-${{ github.ref_name }}-darwin_amd64.zip
            release/darwin-arm64/datadatdat-cli-${{ github.ref_name }}-darwin_arm64.zip
            release/linux-amd64/datadatdat-cli-${{ github.ref_name }}-linux_amd64.tar
            release/linux-arm64/datadatdat-cli-${{ github.ref_name }}-linux_arm64.tar
            release/windows/datadatdat-cli-${{ github.ref_name }}-windows_amd64.zip
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  
  e2e-test:
    needs: build
    runs-on: ubuntu-22.04
    steps:
      - uses: actions/checkout@v4
      
      - name: Download release artifacts
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          gh release download $VERSION --pattern "datadatdat-cli-$VERSION-linux_amd64.tar"
          tar -xf datadatdat-cli-$VERSION-linux_amd64.tar
          chmod +x d3
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Install BATS
        run: sudo npm install -g bats
      
      - name: Install ZFS
        run: |
          sudo apt-get update
          sudo apt-get install -y zfsutils-linux
          sudo modprobe zfs
      
      - name: Run E2E tests
        run: |
          export PATH=$PWD:$PATH
          make e2e
  
  publish:
    needs: e2e-test
    runs-on: ubuntu-latest
    steps:
      - name: Publish release
        run: |
          VERSION=${GITHUB_REF#refs/tags/}
          gh release edit $VERSION --draft=false --latest
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Benefits:**
- Eliminates manual `make release` + artifact uploads
- Automated E2E testing before publish
- Ensures binaries work before release

## Phase 3: Version Validation

### 3.1 Dependency Alignment Check
**Location**: `datadatdat/.github/workflows/validate-dependencies.yml`

```yaml
name: Validate Dependencies
on:
  pull_request:
    paths:
      - 'go.mod'
      - 'go.sum'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Check for version conflicts
        run: |
          # Check all providers use same remote-sdk-go version
          go mod graph | grep datadatdat | grep remote-sdk-go | tee /tmp/sdk-versions.txt
          
          # Count unique versions
          SDK_VERSIONS=$(cat /tmp/sdk-versions.txt | awk '{print $NF}' | sort -u | wc -l)
          
          if [ $SDK_VERSIONS -gt 1 ]; then
            echo "❌ ERROR: Multiple remote-sdk-go versions detected!"
            cat /tmp/sdk-versions.txt
            exit 1
          fi
          
          echo "✅ All providers use same remote-sdk-go version"
      
      - name: Check for replace directives
        run: |
          if grep -q "^replace" go.mod; then
            echo "❌ ERROR: Replace directives found in go.mod!"
            grep "^replace" go.mod
            exit 1
          fi
          echo "✅ No replace directives"
```

**Benefits:**
- Prevents version conflicts before merge
- Catches replace directives before release
- Automated validation in PR process

## Phase 4: Master Release Coordinator (Future)

### 4.1 Orchestrated Release Pipeline
**Location**: `datadatdat/.github/workflows/orchestrate-release.yml`

```yaml
name: Orchestrate Full Release
on:
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to release (e.g., v1.3.0)'
        required: true

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Validate version format
        run: |
          if ! [[ "${{ github.event.inputs.version }}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo "Invalid version format"
            exit 1
          fi
  
  release-sdk:
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - name: Release remote-sdk-go
        run: |
          # Tag and trigger remote-sdk-go release
          gh workflow run release.yml --repo datadatdat/remote-sdk-go --ref master
  
  wait-for-providers:
    needs: release-sdk
    runs-on: ubuntu-latest
    steps:
      - name: Wait for provider PRs
        run: |
          # Monitor for auto-created PRs from cascade workflow
          # Auto-approve and merge when tests pass
  
  release-cli:
    needs: wait-for-providers
    runs-on: ubuntu-latest
    steps:
      - name: Release datadatdat CLI
        run: |
          # Update dependencies and tag
  
  # ... continue for server, remote-server
```

**Benefits:**
- One command releases entire ecosystem
- Enforces correct release order
- Automated waiting between phases
- Reduces human error

## Implementation Timeline for v1.3.0

### Before Release (Week 1-2)
- [ ] Implement Phase 1.1: Dependency cascade workflow
- [ ] Implement Phase 1.2: Auto-publish after tests
- [ ] Implement Phase 2.1: CLI release workflow
- [ ] Implement Phase 3.1: Dependency validation

### During v1.3.0 Release (Week 3)
- [ ] Use new automated workflows
- [ ] Document any issues encountered
- [ ] Measure time saved vs manual process

### After Release (Week 4)
- [ ] Evaluate automation effectiveness
- [ ] Plan Phase 4 implementation
- [ ] Update RELEASE.md with new automated process

## Expected Time Savings

### Current Manual Process: ~4-6 hours
- Phase 1: Foundation (30 min)
- Phase 1.2: Update 5 providers (30 min) ← **AUTOMATED**
- Phase 1.2: Publish 5 releases (15 min) ← **AUTOMATED**
- Phase 2: Kotlin providers (20 min)
- Phase 3: Client (10 min)
- Phase 4: CLI dependencies (20 min) ← **VALIDATED**
- Phase 4: CLI build & release (30 min) ← **AUTOMATED**
- Phase 4: E2E testing (20 min) ← **AUTOMATED**
- Phase 4: Publish (5 min) ← **AUTOMATED**
- Phase 5: Server (15 min)
- Phase 6: Remote server (30 min)
- Phase 7: Validation (30 min)

### With Automation: ~1.5-2 hours
- Manual: Tag SDK, approve provider PRs, validate, tag CLI
- Automated: Everything else

**Time Saved: 2.5-4 hours per release (60-65% reduction)**

## Risk Mitigation

### What Could Go Wrong
1. **Automated PRs create merge conflicts**
   - Mitigation: Stagger provider updates, manual review required

2. **Tests pass in automation but fail locally**
   - Mitigation: Use identical test environments (Ubuntu 22.04)

3. **Auto-publish releases with bugs**
   - Mitigation: Keep E2E testing as manual gate initially

4. **Version conflicts still occur**
   - Mitigation: Validation runs on every PR

## Success Metrics

- [ ] Release time reduced by >50%
- [ ] Zero version conflicts in v1.3.0 release
- [ ] All tests pass in automated pipelines
- [ ] No rollbacks required due to automation issues
