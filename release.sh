#!/bin/bash
#
# Datadatdat v1.0.0 Release Script
# 
# This script executes the complete v1.0.0 release process for all Datadatdat repositories
# following proper dependency order and version formatting requirements.
#
# CRITICAL: Kotlin/Maven repos use "1.0.0" format (NO 'v' prefix)
#          Go repos use "v1.0.0" format (WITH 'v' prefix) for Git tags
#
# Usage: ./release.sh [verify|foundation|providers|infrastructure|core|docker|all]
#

set -e  # Exit on any error

WORKSPACE_ROOT="/c/dev"
VERSION="1.0.0"
GIT_TAG="v1.0.0"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

# Verify pre-release status
verify_versions() {
    log "=== Current Version Status ==="
    
    # Check Go dependencies (should all be v1.0.0)
    log "Checking Go dependencies..."
    for repo in datadatdat datadatdat-server datadatdat-client-go datadatdat-docker-proxy remote-sdk-go s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go; do
        echo "--- $repo ---"
        cd "$WORKSPACE_ROOT/$repo"
        if [ -f go.mod ]; then
            echo "Go dependencies:"
            grep "github.com/datadatdat" go.mod || echo "No datadatdat dependencies"
        fi
    done
    
    # Check Kotlin Maven versions (should all be 1.0.0 - NO 'v' prefix)
    log "Checking Kotlin Maven dependencies..."
    for repo in remote-sdk command-executor plugin-launcher s3-remote ssh-remote s3web-remote nop-remote delphix-remote; do
        echo "--- $repo ---"
        cd "$WORKSPACE_ROOT/$repo"
        if [ -f build.gradle.kts ]; then
            echo "Maven dependencies:"
            grep -E "com\.datadatdat.*[0-9]" build.gradle.kts || echo "No datadatdat Maven deps"
        fi
    done
    
    success "Version verification complete"
}

# Release foundation components
release_foundation() {
    log "=== Phase 1: Foundation Components ==="
    
    # command-executor - Maven artifact (NO 'v' prefix)
    log "Releasing command-executor..."
    cd "$WORKSPACE_ROOT/command-executor"
    ./gradlew publish -Pversion=$VERSION
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "command-executor v1.0.0 released"
    
    # remote-sdk - Maven artifact (NO 'v' prefix)
    log "Releasing remote-sdk..."
    cd "$WORKSPACE_ROOT/remote-sdk"
    ./gradlew publish -Pversion=$VERSION
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "remote-sdk v1.0.0 released"
    
    # remote-sdk-go - Go module
    log "Releasing remote-sdk-go..."
    cd "$WORKSPACE_ROOT/remote-sdk-go"
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "remote-sdk-go v1.0.0 released"
    
    success "Foundation components released"
}

# Release remote providers
release_providers() {
    log "=== Phase 2: Remote Providers ==="
    
    # Kotlin providers - Maven artifacts (NO 'v' prefix for publishing)
    log "Releasing Kotlin remote providers..."
    for repo in s3-remote ssh-remote s3web-remote nop-remote delphix-remote; do
        log "Releasing $repo..."
        cd "$WORKSPACE_ROOT/$repo"
        ./gradlew publish -Pversion=$VERSION
        git tag $GIT_TAG  # Git tags still use 'v' prefix
        git push origin $GIT_TAG
        success "$repo v1.0.0 released"
    done
    
    # Go providers - already at v1.0.0, just release
    log "Releasing Go remote providers..."
    for repo in s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go; do
        log "Releasing $repo..."
        cd "$WORKSPACE_ROOT/$repo"
        git tag $GIT_TAG
        git push origin $GIT_TAG
        success "$repo v1.0.0 released"
    done
    
    success "Remote providers released"
}

# Release plugin infrastructure
release_infrastructure() {
    log "=== Phase 3: Plugin Infrastructure ==="
    
    log "Releasing plugin-launcher..."
    cd "$WORKSPACE_ROOT/plugin-launcher"
    ./gradlew publish -Pversion=$VERSION
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "plugin-launcher v1.0.0 released"
    
    success "Plugin infrastructure released"
}

# Release client and CLI
release_core() {
    log "=== Phase 4: Client and CLI ==="
    
    # Client - already at v1.0.0
    log "Releasing datadatdat-client-go..."
    cd "$WORKSPACE_ROOT/datadatdat-client-go"
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "datadatdat-client-go v1.0.0 released"
    
    # CLI - need to update dependencies to use released versions first
    log "Updating CLI dependencies to released versions..."
    cd "$WORKSPACE_ROOT/datadatdat"
    
    # Update dependencies to the newly released v1.0.0 versions
    log "Updating Go dependencies..."
    go get github.com/datadatdat/remote-sdk-go@$GIT_TAG
    go get github.com/datadatdat/s3-remote-go@$GIT_TAG
    go get github.com/datadatdat/ssh-remote-go@$GIT_TAG
    go get github.com/datadatdat/s3web-remote-go@$GIT_TAG
    go get github.com/datadatdat/nop-remote-go@$GIT_TAG
    go get github.com/datadatdat/datadatdat-client-go@$GIT_TAG
    go mod tidy
    
    # Build CLI with proper version injection
    log "Building CLI with version $VERSION..."
    export VERSION=$VERSION
    make clean
    make release
    
    # Commit dependency updates
    git add go.mod go.sum
    git commit -m "Update dependencies to v1.0.0"
    
    # Create git tag and push
    git tag $GIT_TAG
    git push origin master
    git push origin $GIT_TAG
    success "datadatdat CLI v1.0.0 released with proper version"
    
    success "Client and CLI released"
}

# Release server and Docker components
release_docker() {
    log "=== Phase 5: Server and Docker ==="
    
    # Docker proxy - already at v1.0.0 dependencies
    log "Releasing datadatdat-docker-proxy..."
    cd "$WORKSPACE_ROOT/datadatdat-docker-proxy"
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "datadatdat-docker-proxy v1.0.0 released"
    
    # Server - will create datadatdat/datadatdat:1.0.0 Docker image
    log "Releasing datadatdat-server..."
    log "This will trigger GitHub Actions to publish datadatdat/datadatdat:1.0.0 to DockerHub"
    cd "$WORKSPACE_ROOT/datadatdat-server"
    git tag $GIT_TAG
    git push origin $GIT_TAG
    success "datadatdat-server v1.0.0 released (Docker publishing triggered)"
    
    success "Server and Docker components released"
}

# Verify post-release status
verify_release() {
    log "=== Post-Release Verification ==="
    
    # Verify all Git tags
    log "Verifying Git tags..."
    for repo in datadatdat datadatdat-server datadatdat-client-go datadatdat-docker-proxy remote-sdk-go s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go command-executor remote-sdk plugin-launcher s3-remote ssh-remote s3web-remote nop-remote delphix-remote; do
        echo "--- $repo ---"
        cd "$WORKSPACE_ROOT/$repo"
        if git tag | grep -q "$GIT_TAG"; then
            success "$repo has v1.0.0 tag"
        else
            warning "$repo missing v1.0.0 tag"
        fi
    done
    
    # Verify Docker image (may take time to publish)
    log "Checking Docker image availability..."
    if docker pull datadatdat/datadatdat:1.0.0 2>/dev/null; then
        success "Docker image datadatdat/datadatdat:1.0.0 available"
        docker inspect datadatdat/datadatdat:1.0.0 > /dev/null
        success "Docker image inspection passed"
    else
        warning "Docker image may still be publishing (check GitHub Actions)"
    fi
    
    # Verify CLI version
    log "Verifying CLI version..."
    cd "$WORKSPACE_ROOT/datadatdat"
    
    # Test version injection
    log "Testing version injection..."
    export VERSION=$VERSION
    if make build && ./build/d3 --version | grep -q "$VERSION"; then
        success "CLI version correctly shows $VERSION"
    else
        warning "CLI version check failed - version may not be properly injected"
        # Try alternative version check
        if go run -ldflags "-X datadatdat/internal/app/commands.Version=$VERSION" cmd/datadatdat/datadatdat.go --version 2>/dev/null | grep -q "$VERSION"; then
            success "CLI version injection works with direct build"
        else
            warning "CLI version injection failed - manual verification needed"
        fi
    fi
    
    success "Release verification complete"
}

# Main execution
case "${1:-all}" in
    verify)
        verify_versions
        ;;
    foundation)
        release_foundation
        ;;
    providers)
        release_providers
        ;;
    infrastructure)
        release_infrastructure
        ;;
    core)
        release_core
        ;;
    docker)
        release_docker
        ;;
    all)
        log "Starting complete v1.0.0 release process..."
        verify_versions
        echo
        release_foundation
        echo
        release_providers
        echo
        release_infrastructure
        echo
        release_core
        echo
        release_docker
        echo
        verify_release
        echo
        success "🎉 Complete v1.0.0 release process finished!"
        ;;
    *)
        echo "Usage: $0 [verify|foundation|providers|infrastructure|core|docker|all]"
        echo
        echo "Commands:"
        echo "  verify         - Check current version status"
        echo "  foundation     - Release foundation components (command-executor, remote-sdk, remote-sdk-go)"
        echo "  providers      - Release remote providers (Kotlin and Go)"
        echo "  infrastructure - Release plugin infrastructure"
        echo "  core           - Release client and CLI"
        echo "  docker         - Release server and Docker components"
        echo "  all            - Execute complete release process (default)"
        exit 1
        ;;
esac