#!/usr/bin/env bash
#
# Dit Ecosystem Release Automation
#
# Automates the full multi-repo release process for all dit components.
# Each repo has GitHub Actions that build/test/publish on tag push - this script
# orchestrates the dependency order, waits for CI, merges PRs, and verifies artifacts.
#
# Usage:
#   ./release.sh VERSION [OPTIONS]
#
#   VERSION        Target version without 'v' prefix (e.g., 1.6.1)
#
# Options:
#   --from-phase N   Resume from phase N (skip earlier phases)
#   --phase N        Run only phase N
#   --dry-run        Show commands without executing
#   --skip-ecs       Skip ECS deployment (Phase 8)
#   --prev VERSION   Previous version to replace (default: auto-detected from go.mod)
#   --timeout SECS   Workflow wait timeout in seconds (default: 900)
#
# Phases:
#   0  Pre-flight checks
#   1  Go foundation (remote-sdk-go + 5 Go providers)
#   2  Kotlin foundation (command-executor + remote-sdk + 6 Kotlin providers)
#   3  dit-client-go
#   4  dit-docker-proxy
#   5  dit-server
#   6  dit CLI
#   7  dit-remote-server
#   8  AWS ECS production deployment
#   9  Post-release validation

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

# WORKSPACE is the directory containing all the cloned repos. Each repo must be
# checked out under its post-rename name (dit, dit-server, dit-remote-go,
# dit-client-go, dit-docker-proxy, dit-remote-server, remote-sdk-go, ...).
# Override with DIT_WORKSPACE if your checkout lives elsewhere.
WORKSPACE="${DIT_WORKSPACE:-/c/dev/dit}"
ORG="ditdotdev"
MAVEN_BUCKET="dit-maven"
PROD_RELEASES_BUCKET="dit-releases-prod"
DEV_RELEASES_BUCKET="dit-releases"
DEV_MINIO_ENDPOINT="localhost:9000"
ECR_REGION="us-west-2"
ECS_CLUSTER="dit-prod"
STATE_DIR="$WORKSPACE/dit/.release-state"

GO_PROVIDERS=(s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go dit-remote-go)
KOTLIN_PROVIDERS=(s3-remote ssh-remote s3web-remote nop-remote delphix-remote dit-remote)
COMMAND_EXECUTOR_DEPENDENTS=(ssh-remote delphix-remote)

GO_MOD_REPOS=(dit dit-remote-go dit-remote-server nop-remote-go s3-remote-go s3web-remote-go ssh-remote-go)

ECR_SERVICES=(auth-server api-gateway api-repo-manifest api-ingest api-download dit-repo-web worker web)
PROD_URL="https://dit.dev"

# Go module-fetch settings for the private github.com/ditdotdev/* modules.
# EXPORTED (not `go env -w`) so they survive the go-toolchain auto-switch
# (go.mod pins go 1.26.2): the switched toolchain otherwise bypasses the
# env-file GOPRIVATE, so `go get` consults sum.golang.org for a private module
# -> 404, surfacing as a misleading "could not read Username" git-auth error
# (ditdotdev/dit#161). GOSUMDB=off disables the checksum-DB lookup entirely.
export GOPRIVATE='github.com/ditdotdev/*'
export GOSUMDB=off
export GOPROXY=direct

# ============================================================================
# Globals (set by parse_args)
# ============================================================================

VERSION=""
V_VERSION=""          # "v1.6.1"
PREV_VERSION=""
PREV_V_VERSION=""     # "v1.6.0"
DRY_RUN=false
SKIP_ECS=false
FROM_PHASE=0
SINGLE_PHASE=-1
WORKFLOW_TIMEOUT=1800  # 30 minutes
POLL_INTERVAL=15      # seconds between polls

# ============================================================================
# Logging
# ============================================================================

log_info()    { echo "[$(date +%H:%M:%S)] $*"; }
log_success() { echo "[$(date +%H:%M:%S)] [OK] $*"; }
log_warn()    { echo "[$(date +%H:%M:%S)] [WARN] $*"; }
log_error()   { echo "[$(date +%H:%M:%S)] [ERROR] $*"; }
log_phase()   { echo ""; echo "========================================"; echo "  Phase $1: $2"; echo "========================================"; echo ""; }
log_step()    { echo "  -> $*"; }
log_dry()     { echo "  [DRY-RUN] $*"; }

# ============================================================================
# Utility Functions
# ============================================================================

run_cmd() {
    # Execute a command, or print it in dry-run mode
    if $DRY_RUN; then
        log_dry "$*"
        return 0
    fi
    "$@"
}

cleanup_draft_release() {
    # Delete any existing draft release for a tag to prevent upload conflicts.
    # The draft-release.yml workflow (release-drafter) creates drafts on master push,
    # which can interfere with the release.yml upload step.
    local repo="$1"
    local tag="$2"
    if $DRY_RUN; then
        return 0
    fi
    if gh release view "$tag" --repo "$ORG/$repo" --json isDraft --jq '.isDraft' 2>/dev/null | grep -q "true"; then
        log_warn "Deleting stale draft release $tag in $ORG/$repo"
        gh release delete "$tag" --repo "$ORG/$repo" --yes 2>/dev/null || true
    fi
}

tag_and_push() {
    # Usage: tag_and_push REPO_PATH TAG
    local repo_path="$1"
    local tag="$2"
    local repo_name
    repo_name=$(basename "$repo_path")
    log_step "Tagging $repo_path at $tag"
    if $DRY_RUN; then
        log_dry "cd $repo_path && git tag $tag && git push origin $tag"
        return 0
    fi
    # Clean up any draft releases that could conflict with upload
    cleanup_draft_release "$repo_name" "$tag"
    cd "$repo_path"
    git tag "$tag"
    git push origin "$tag"
    # Wait briefly for draft-release.yml (release-drafter) to possibly create a new draft,
    # then clean it up before the release.yml workflow reaches the upload step
    sleep 10
    cleanup_draft_release "$repo_name" "$tag"
}

commit_and_push() {
    # Usage: commit_and_push REPO_PATH MESSAGE [FILES...]
    local repo_path="$1"
    local message="$2"
    shift 2
    local files=("$@")
    log_step "Committing in $repo_path: $message"
    if $DRY_RUN; then
        log_dry "cd $repo_path && git add ${files[*]} && git commit -m '$message' && git push origin master"
        return 0
    fi
    cd "$repo_path"
    git add "${files[@]}"
    if git diff --cached --quiet; then
        log_step "No changes to commit in $repo_path"
        return 0
    fi
    git commit -m "$message"
    git push origin master
}

update_go_dep() {
    # Usage: update_go_dep REPO_PATH MODULE VERSION
    local repo_path="$1"
    local module="$2"
    local version="$3"
    log_step "Updating $module to $version in $repo_path"
    if $DRY_RUN; then
        log_dry "cd $repo_path && go get $module@$version && go mod tidy"
        return 0
    fi
    cd "$repo_path"
    # GOPRIVATE / GOSUMDB / GOPROXY are exported at the top of the script.
    go get "$module@$version"
    go mod tidy
}

wait_for_workflow() {
    # Usage: wait_for_workflow REPO [WORKFLOW_FILE]
    # Waits for the most recent workflow run to complete
    local repo="$1"
    local workflow="${2:-release.yml}"
    local elapsed=0

    if $DRY_RUN; then
        log_dry "Wait for workflow $workflow in $ORG/$repo"
        return 0
    fi

    log_step "Waiting for $workflow in $ORG/$repo..."
    sleep 5  # Give GitHub a moment to register the run

    while [ $elapsed -lt $WORKFLOW_TIMEOUT ]; do
        local status conclusion
        status=$(gh run list --repo "$ORG/$repo" --workflow="$workflow" --limit 1 --json status,conclusion --jq '.[0].status' 2>/dev/null || echo "unknown")
        conclusion=$(gh run list --repo "$ORG/$repo" --workflow="$workflow" --limit 1 --json status,conclusion --jq '.[0].conclusion' 2>/dev/null || echo "")

        if [ "$status" = "completed" ]; then
            if [ "$conclusion" = "success" ]; then
                log_success "Workflow $workflow in $ORG/$repo completed successfully"
                return 0
            else
                log_error "Workflow $workflow in $ORG/$repo failed with conclusion: $conclusion"
                log_error "Check: gh run list --repo $ORG/$repo --workflow=$workflow --limit 1"
                return 1
            fi
        fi

        sleep "$POLL_INTERVAL"
        elapsed=$((elapsed + POLL_INTERVAL))
        printf "    ... Waiting (%ds / %ds) status=%s\r" "$elapsed" "$WORKFLOW_TIMEOUT" "$status"
    done
    echo ""
    log_error "Timeout waiting for $workflow in $ORG/$repo after ${WORKFLOW_TIMEOUT}s"
    return 1
}

wait_for_pr() {
    # Usage: wait_for_pr REPO BRANCH_PATTERN
    # Returns the PR number
    local repo="$1"
    local branch_pattern="$2"
    local elapsed=0

    if $DRY_RUN; then
        log_dry "Wait for PR matching '$branch_pattern' in $ORG/$repo" >&2
        echo "0"
        return 0
    fi

    log_step "Waiting for PR matching '$branch_pattern' in $ORG/$repo..." >&2
    while [ $elapsed -lt $WORKFLOW_TIMEOUT ]; do
        local pr_number
        pr_number=$(gh pr list --repo "$ORG/$repo" --head "$branch_pattern" --json number --jq '.[0].number' 2>/dev/null || echo "")

        if [ -n "$pr_number" ] && [ "$pr_number" != "null" ]; then
            log_success "Found PR #$pr_number in $ORG/$repo" >&2
            echo "$pr_number"
            return 0
        fi

        sleep "$POLL_INTERVAL"
        elapsed=$((elapsed + POLL_INTERVAL))
    done
    log_error "Timeout waiting for PR in $ORG/$repo matching '$branch_pattern'" >&2
    return 1
}

wait_for_pr_checks() {
    # Usage: wait_for_pr_checks REPO PR_NUMBER
    local repo="$1"
    local pr_number="$2"
    local elapsed=0

    if $DRY_RUN; then
        log_dry "Wait for PR #$pr_number checks in $ORG/$repo"
        return 0
    fi

    log_step "Waiting for PR #$pr_number checks in $ORG/$repo..."
    sleep 10  # Give checks time to start

    while [ $elapsed -lt $WORKFLOW_TIMEOUT ]; do
        local all_passed=true
        local any_pending=false

        while IFS=$'\t' read -r name state; do
            if [ "$state" = "SUCCESS" ] || [ "$state" = "SKIPPED" ] || [ "$state" = "NEUTRAL" ]; then
                : # passed
            elif [ "$state" = "FAILURE" ] || [ "$state" = "ERROR" ]; then
                log_error "Check '$name' failed in PR #$pr_number ($ORG/$repo): $state"
                return 1
            else
                any_pending=true
                all_passed=false
            fi
        done < <(gh pr checks "$pr_number" --repo "$ORG/$repo" --json name,state --jq '.[] | [.name, .state] | @tsv' 2>/dev/null || echo "")

        if $all_passed && ! $any_pending; then
            log_success "All checks passed for PR #$pr_number in $ORG/$repo"
            return 0
        fi

        sleep "$POLL_INTERVAL"
        elapsed=$((elapsed + POLL_INTERVAL))
    done
    log_error "Timeout waiting for PR checks in $ORG/$repo"
    return 1
}

verify_s3_artifact() {
    # Usage: verify_s3_artifact BUCKET PATH
    local bucket="$1"
    local path="$2"

    if $DRY_RUN; then
        log_dry "Verify S3 artifact: s3://$bucket/$path"
        return 0
    fi

    if aws s3 ls "s3://$bucket/$path" > /dev/null 2>&1; then
        log_success "Verified artifact: s3://$bucket/$path"
        return 0
    else
        log_error "Artifact NOT found: s3://$bucket/$path"
        return 1
    fi
}

verify_maven_pom() {
    # Usage: verify_maven_pom ARTIFACT VERSION DEP_ARTIFACT DEP_VERSION
    # Downloads POM from S3 Maven bucket and checks that a dependency has the expected version.
    # Catches stale transitive deps (e.g., command-executor:1.6.0 in remote-sdk:1.6.1).
    local artifact="$1"
    local version="$2"
    local dep_artifact="$3"
    local dep_version="$4"

    if $DRY_RUN; then
        log_dry "Verify POM: $artifact:$version contains $dep_artifact:$dep_version"
        return 0
    fi

    local pom_path="dev/dit/$artifact/$version/$artifact-$version.pom"
    local tmp_pom
    tmp_pom=$(mktemp)
    if aws s3 cp "s3://$MAVEN_BUCKET/$pom_path" "$tmp_pom" 2>/dev/null; then
        if grep -q "<artifactId>$dep_artifact</artifactId>" "$tmp_pom"; then
            local pom_dep_version
            pom_dep_version=$(grep -A1 "<artifactId>$dep_artifact</artifactId>" "$tmp_pom" | grep "<version>" | sed 's/.*<version>\(.*\)<\/version>.*/\1/')
            if [ "$pom_dep_version" = "$dep_version" ]; then
                log_success "POM verified: $artifact:$version contains $dep_artifact:$dep_version"
            else
                log_error "POM MISMATCH: $artifact:$version has $dep_artifact:$pom_dep_version (expected $dep_version)"
                rm -f "$tmp_pom"
                return 1
            fi
        else
            log_info "POM for $artifact:$version does not reference $dep_artifact (OK if not a dependency)"
        fi
    else
        log_warn "Could not download POM: s3://$MAVEN_BUCKET/$pom_path"
    fi
    rm -f "$tmp_pom"
}

verify_gh_release() {
    # Usage: verify_gh_release REPO TAG
    local repo="$1"
    local tag="$2"

    if $DRY_RUN; then
        log_dry "Verify GitHub release $tag in $ORG/$repo"
        return 0
    fi

    if gh release view "$tag" --repo "$ORG/$repo" > /dev/null 2>&1; then
        log_success "Verified release: $ORG/$repo@$tag"
        return 0
    else
        log_error "Release NOT found: $ORG/$repo@$tag"
        return 1
    fi
}

save_phase_state() {
    local phase="$1"
    mkdir -p "$STATE_DIR"
    echo "$phase" > "$STATE_DIR/$VERSION"
}

get_completed_phase() {
    if [ -f "$STATE_DIR/$VERSION" ]; then
        cat "$STATE_DIR/$VERSION"
    else
        echo "-1"
    fi
}

detect_prev_version() {
    # Auto-detect previous version from go.mod
    local prev
    prev=$(grep 'github.com/ditdotdev/remote-sdk-go' "$WORKSPACE/dit/go.mod" | awk '{print $2}' | sed 's/^v//')
    echo "$prev"
}

# ============================================================================
# Phase 0: Pre-flight Checks
# ============================================================================

phase_preflight() {
    log_phase 0 "Pre-flight Checks"

    # Check tools
    log_step "Checking required tools..."
    for tool in gh aws go git; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is not installed or not in PATH"
            exit 1
        fi
    done
    log_success "All required tools available"

    # Check gh auth
    log_step "Checking GitHub authentication..."
    if ! gh auth status &> /dev/null; then
        log_error "gh CLI not authenticated. Run: gh auth login"
        exit 1
    fi
    log_success "GitHub CLI authenticated"

    # Check AWS credentials
    log_step "Checking AWS credentials..."
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS CLI not configured. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY"
        exit 1
    fi
    log_success "AWS credentials valid"

    # Check all repos exist and are on master with clean state.
    #
    # The clean-tree check is load-bearing for dit-remote-server in
    # particular: Phase 8 step 0 runs `terraform apply` against that repo's
    # deploy/terraform/ directory (issue #137). An uncommitted change in
    # that subtree would mean what we apply diverges from what's on master
    # — exactly the drift the new step is meant to PREVENT.
    log_step "Checking repository states..."
    local all_repos=(
        remote-sdk-go "${GO_PROVIDERS[@]}"
        command-executor remote-sdk "${KOTLIN_PROVIDERS[@]}"
        dit-client-go dit-docker-proxy dit-server
        dit dit-remote-server
    )
    for repo in "${all_repos[@]}"; do
        local repo_path="$WORKSPACE/$repo"
        if [ ! -d "$repo_path" ]; then
            log_error "Repository not found: $repo_path"
            exit 1
        fi
        local branch
        branch=$(cd "$repo_path" && git rev-parse --abbrev-ref HEAD)
        if [ "$branch" != "master" ]; then
            log_warn "$repo is on branch '$branch', switching to master"
            if ! $DRY_RUN; then
                cd "$repo_path" && git checkout master && git pull origin master
            fi
        fi
        local dirty
        dirty=$(cd "$repo_path" && git status --porcelain)
        if [ -n "$dirty" ]; then
            if $DRY_RUN; then
                log_warn "$repo has uncommitted changes (ignored in dry-run):\n$dirty"
            else
                log_error "$repo has uncommitted changes:\n$dirty"
                exit 1
            fi
        fi
    done
    log_success "All repositories on master with clean working trees"

    # Pull latest on all repos
    log_step "Pulling latest on all repos..."
    if ! $DRY_RUN; then
        for repo in "${all_repos[@]}"; do
            cd "$WORKSPACE/$repo" && git pull origin master 2>/dev/null || true
        done
    fi
    log_success "All repos up to date"

    # Check for replace directives
    log_step "Checking for replace directives in go.mod files..."
    local has_replace=false
    for repo in "${GO_MOD_REPOS[@]}"; do
        local gomod="$WORKSPACE/$repo/go.mod"
        if [ -f "$gomod" ] && grep -q "^replace" "$gomod" 2>/dev/null; then
            log_warn "Found replace directives in $repo/go.mod - removing..."
            if ! $DRY_RUN; then
                cd "$WORKSPACE/$repo"
                sed -i '/^replace/d' go.mod
                go mod tidy
                git add go.mod go.sum
                git commit -m "Remove replace directives for v$VERSION release" || true
                git push origin master
            fi
            has_replace=true
        fi
    done
    if ! $has_replace; then
        log_success "No replace directives found"
    fi

    # Check that target version tags don't already exist
    log_step "Checking for existing v$VERSION tags..."
    for repo in "${all_repos[@]}"; do
        cd "$WORKSPACE/$repo"
        if git tag -l "v$VERSION" | grep -q "v$VERSION" 2>/dev/null; then
            log_warn "Tag v$VERSION already exists in $repo"
        fi
        if git tag -l "$VERSION" | grep -q "$VERSION" 2>/dev/null; then
            log_warn "Tag $VERSION already exists in $repo"
        fi
    done

    save_phase_state 0
    log_success "Pre-flight checks complete"
}

# ============================================================================
# Phase 1: Go Foundation
# ============================================================================

phase_go_foundation() {
    log_phase 1 "Go Foundation (remote-sdk-go + 5 Go providers)"

    # 1. Tag remote-sdk-go
    tag_and_push "$WORKSPACE/remote-sdk-go" "v$VERSION"

    # 2. Wait for SDK release workflow
    wait_for_workflow "remote-sdk-go" "release.yml"

    # 3. Wait for auto-PRs in all 5 provider repos
    log_step "Waiting for auto-PRs in provider repos..."
    declare -A PR_NUMBERS
    for provider in "${GO_PROVIDERS[@]}"; do
        local pr_num
        pr_num=$(wait_for_pr "$provider" "auto-update-sdk-v$VERSION")
        PR_NUMBERS[$provider]=$pr_num
    done
    log_success "All 5 provider PRs created"

    # 4. Wait for PR checks to pass
    for provider in "${GO_PROVIDERS[@]}"; do
        wait_for_pr_checks "$provider" "${PR_NUMBERS[$provider]}"
    done
    log_success "All provider PR checks passed"

    # 5. Merge all PRs
    log_step "Merging provider PRs..."
    for provider in "${GO_PROVIDERS[@]}"; do
        if $DRY_RUN; then
            log_dry "gh pr merge ${PR_NUMBERS[$provider]} --repo $ORG/$provider --squash --delete-branch"
        else
            gh pr merge "${PR_NUMBERS[$provider]}" --repo "$ORG/$provider" --squash --delete-branch
            log_success "Merged PR #${PR_NUMBERS[$provider]} in $provider"
        fi
    done

    # 6. Pull and tag each provider
    log_step "Tagging providers at v$VERSION..."
    for provider in "${GO_PROVIDERS[@]}"; do
        if ! $DRY_RUN; then
            cd "$WORKSPACE/$provider"
            git pull origin master
        fi
        tag_and_push "$WORKSPACE/$provider" "v$VERSION"
    done

    # 7. Wait for all provider release workflows
    for provider in "${GO_PROVIDERS[@]}"; do
        wait_for_workflow "$provider" "release.yml"
    done

    # 8. Verify provider alignment — all must resolve to same remote-sdk-go version
    if ! $DRY_RUN; then
        log_step "Verifying Go provider alignment..."
        cd "$WORKSPACE/dit"
        for provider in "${GO_PROVIDERS[@]}"; do
            go get "github.com/ditdotdev/$provider@v$VERSION" 2>/dev/null || true
        done
        go mod tidy 2>/dev/null || true
        local sdk_versions
        sdk_versions=$(go mod graph | awk '$2 ~ /dit\/remote-sdk-go@/ {print $2}' | sort -u)
        local sdk_count
        sdk_count=$(echo "$sdk_versions" | grep -c . || echo "0")
        if [ "$sdk_count" -gt 1 ]; then
            log_error "Provider alignment FAILED! Multiple remote-sdk-go versions detected:"
            echo "$sdk_versions"
            log_error "STOP: Fix provider alignment before continuing to Phase 2"
            exit 1
        fi
        log_success "All providers aligned on: $sdk_versions"
        # Restore go.mod — Phase 6 will do the real update
        git checkout go.mod go.sum 2>/dev/null || true
    fi

    save_phase_state 1
    log_success "Phase 1 complete: Go foundation released at v$VERSION"
}

# ============================================================================
# Phase 2: Kotlin Foundation
# ============================================================================

phase_kotlin_foundation() {
    log_phase 2 "Kotlin Foundation (command-executor + remote-sdk + 6 Kotlin providers)"

    # 1. Tag and release command-executor
    tag_and_push "$WORKSPACE/command-executor" "$VERSION"
    wait_for_workflow "command-executor" "release.yml"
    verify_s3_artifact "$MAVEN_BUCKET" "dev/dit/command-executor/$VERSION/"

    # Update command-executor dependency in all downstream repos BEFORE tagging them
    log_step "Updating command-executor dependency to $VERSION in downstream repos..."
    local ce_downstream_files=(
        "$WORKSPACE/remote-sdk/build.gradle.kts"
        "$WORKSPACE/dit-server/server/build.gradle.kts"
        "$WORKSPACE/ssh-remote/server/build.gradle.kts"
        "$WORKSPACE/delphix-remote/server/build.gradle.kts"
    )
    for gradle_file in "${ce_downstream_files[@]}"; do
        if [ -f "$gradle_file" ]; then
            if $DRY_RUN; then
                log_dry "Update command-executor -> $VERSION in $gradle_file"
            else
                sed -i "s/command-executor:[0-9][0-9.]*/command-executor:${VERSION}/g" "$gradle_file"
                local repo_dir
                repo_dir=$(dirname "$gradle_file")
                local relative_file
                relative_file=$(basename "$gradle_file")
                # For remote-sdk, build.gradle.kts is in the root
                # For others, it's in server/ so go up one level
                if [[ "$gradle_file" == */server/build.gradle.kts ]]; then
                    repo_dir=$(dirname "$repo_dir")
                    relative_file="server/build.gradle.kts"
                fi
                cd "$repo_dir"
                git add "$relative_file"
                git commit -m "Update command-executor to $VERSION" || true
                git push origin master
                log_success "Updated command-executor in $(basename "$repo_dir")"
            fi
        fi
    done

    # 2. Tag remote-sdk (Kotlin)
    tag_and_push "$WORKSPACE/remote-sdk" "$VERSION"
    wait_for_workflow "remote-sdk" "release.yml"

    # 3. Verify remote-sdk published to Maven (existence + POM content)
    verify_s3_artifact "$MAVEN_BUCKET" "dev/dit/remote-sdk/$VERSION/"
    verify_maven_pom "remote-sdk" "$VERSION" "command-executor" "$VERSION"

    # 4. Update build.gradle.kts in all 6 Kotlin providers
    log_step "Updating Kotlin provider dependencies to $VERSION..."
    for provider in "${KOTLIN_PROVIDERS[@]}"; do
        local repo_path="$WORKSPACE/$provider"
        local files_changed=()

        if $DRY_RUN; then
            log_dry "Update remote-sdk version in $provider/server/build.gradle.kts and client/build.gradle.kts"
        else
            # Update server/build.gradle.kts
            if [ -f "$repo_path/server/build.gradle.kts" ]; then
                sed -i "s/remote-sdk:[0-9][0-9.]*/remote-sdk:${VERSION}/g" "$repo_path/server/build.gradle.kts"
                files_changed+=("server/build.gradle.kts")
            fi
            # Update client/build.gradle.kts
            if [ -f "$repo_path/client/build.gradle.kts" ]; then
                sed -i "s/remote-sdk:[0-9][0-9.]*/remote-sdk:${VERSION}/g" "$repo_path/client/build.gradle.kts"
                files_changed+=("client/build.gradle.kts")
            fi

            # Update command-executor dep for ssh-remote and delphix-remote
            for ce_dep in "${COMMAND_EXECUTOR_DEPENDENTS[@]}"; do
                if [ "$provider" = "$ce_dep" ]; then
                    sed -i "s/command-executor:[0-9][0-9.]*/command-executor:${VERSION}/g" "$repo_path/server/build.gradle.kts"
                fi
            done

            if [ ${#files_changed[@]} -gt 0 ]; then
                commit_and_push "$repo_path" "Update dependencies to $VERSION" "${files_changed[@]}"
            fi
        fi
    done

    # 5. Tag all Kotlin providers (no 'v' prefix for Maven)
    for provider in "${KOTLIN_PROVIDERS[@]}"; do
        tag_and_push "$WORKSPACE/$provider" "$VERSION"
    done

    # 6. Wait for Maven publish workflows
    for provider in "${KOTLIN_PROVIDERS[@]}"; do
        wait_for_workflow "$provider" "release.yml"
    done

    # 7. Verify Maven artifacts (existence + POM content)
    log_step "Verifying Maven artifacts..."
    for provider in "${KOTLIN_PROVIDERS[@]}"; do
        # Server artifact
        verify_s3_artifact "$MAVEN_BUCKET" "dev/dit/${provider}-server/$VERSION/" || \
        verify_s3_artifact "$MAVEN_BUCKET" "dev/dit/${provider}/$VERSION/" || true
        # Verify POM references correct remote-sdk version
        verify_maven_pom "${provider}-server" "$VERSION" "remote-sdk" "$VERSION" || \
        verify_maven_pom "${provider}" "$VERSION" "remote-sdk" "$VERSION" || true
    done

    # 8. Update and tag plugin-launcher (Kotlin + Go, publishes to Maven)
    log_step "Updating plugin-launcher remote-sdk-go dependency..."
    local pl_path="$WORKSPACE/plugin-launcher"
    if [ -d "$pl_path" ]; then
        if ! $DRY_RUN; then
            cd "$pl_path"
            # Update Go dependency (GOPRIVATE/GOSUMDB/GOPROXY exported at top)
            go get "github.com/ditdotdev/remote-sdk-go@v$VERSION"
            go mod tidy
            if [ -n "$(git status --porcelain)" ]; then
                commit_and_push "$pl_path" "Update remote-sdk-go to v$VERSION" go.mod go.sum
            fi
        fi
        tag_and_push "$pl_path" "$VERSION"
        wait_for_workflow "plugin-launcher" "release.yml"
        verify_s3_artifact "$MAVEN_BUCKET" "dev/dit/plugin-launcher/$VERSION/"
        log_success "plugin-launcher released at $VERSION"
    else
        log_warn "plugin-launcher not found at $pl_path - skipping"
    fi

    save_phase_state 2
    log_success "Phase 2 complete: Kotlin foundation released at $VERSION"
}

# ============================================================================
# Phase 3: Client
# ============================================================================

phase_client() {
    log_phase 3 "dit-client-go"

    tag_and_push "$WORKSPACE/dit-client-go" "v$VERSION"
    wait_for_workflow "dit-client-go" "release.yml"
    verify_gh_release "dit-client-go" "v$VERSION"

    save_phase_state 3
    log_success "Phase 3 complete: dit-client-go released at v$VERSION"
}

# ============================================================================
# Phase 4: Docker Proxy
# ============================================================================

phase_docker_proxy() {
    log_phase 4 "dit-docker-proxy"

    local repo_path="$WORKSPACE/dit-docker-proxy"

    # Update client-go dependency
    update_go_dep "$repo_path" "github.com/ditdotdev/dit-client-go" "v$VERSION"

    # Commit and push
    if ! $DRY_RUN; then
        cd "$repo_path"
        if [ -n "$(git status --porcelain)" ]; then
            commit_and_push "$repo_path" "Update dit-client-go to v$VERSION" go.mod go.sum
        fi
    fi

    # Tag and release
    tag_and_push "$repo_path" "v$VERSION"
    wait_for_workflow "dit-docker-proxy" "release.yml"

    # Verify binary uploaded to S3
    verify_s3_artifact "$MAVEN_BUCKET" "dit-docker-proxy/docker-volume-proxy"

    save_phase_state 4
    log_success "Phase 4 complete: dit-docker-proxy released at v$VERSION"
}

# ============================================================================
# Phase 5: Server
# ============================================================================

phase_server() {
    log_phase 5 "dit-server"

    local repo_path="$WORKSPACE/dit-server"
    local gradle_file="$repo_path/server/build.gradle.kts"

    # Update all Kotlin dependencies in build.gradle.kts
    log_step "Updating dit-server dependencies..."
    if ! $DRY_RUN; then
        # Only update command-executor if it was actually released at $VERSION
        if aws s3 ls "s3://$MAVEN_BUCKET/dev/dit/command-executor/$VERSION/" > /dev/null 2>&1; then
            sed -i "s/command-executor:[0-9][0-9.]*/command-executor:${VERSION}/g" "$gradle_file"
        else
            log_info "command-executor:$VERSION not in Maven — keeping existing version"
        fi
        sed -i "s/remote-sdk:[0-9][0-9.]*/remote-sdk:${VERSION}/g" "$gradle_file"
        sed -i "s/dit-remote-client:[0-9][0-9.]*/dit-remote-client:${VERSION}/g" "$gradle_file"
        sed -i "s/dit-remote-server:[0-9][0-9.]*/dit-remote-server:${VERSION}/g" "$gradle_file"
        sed -i "s/nop-remote-server:[0-9][0-9.]*/nop-remote-server:${VERSION}/g" "$gradle_file"
        sed -i "s/ssh-remote-server:[0-9][0-9.]*/ssh-remote-server:${VERSION}/g" "$gradle_file"
        sed -i "s/s3-remote-server:[0-9][0-9.]*/s3-remote-server:${VERSION}/g" "$gradle_file"
        sed -i "s/s3web-remote-server:[0-9][0-9.]*/s3web-remote-server:${VERSION}/g" "$gradle_file"
    fi

    # Update Go dependency
    update_go_dep "$repo_path" "github.com/ditdotdev/dit-client-go" "v$VERSION"

    # Commit and push
    if ! $DRY_RUN; then
        cd "$repo_path"
        if [ -n "$(git status --porcelain)" ]; then
            commit_and_push "$repo_path" "Update dependencies to $VERSION" server/build.gradle.kts go.mod go.sum
        fi
    fi

    # Verify deps updated (at least 6; 7 if command-executor was also released)
    if ! $DRY_RUN; then
        local dep_count
        dep_count=$(grep -c "dev.dit:.*:${VERSION}" "$gradle_file" || echo "0")
        if [ "$dep_count" -lt 6 ]; then
            log_error "Only $dep_count dit dependencies updated in $gradle_file (expected at least 6)"
            exit 1
        fi
        log_success "Verified $dep_count Kotlin dependencies at $VERSION"
    fi

    # Tag and release
    tag_and_push "$repo_path" "v$VERSION"
    wait_for_workflow "dit-server" "release.yml"

    save_phase_state 5
    log_success "Phase 5 complete: dit-server released at v$VERSION"
}

# ============================================================================
# Phase 6: CLI
# ============================================================================

phase_cli() {
    log_phase 6 "dit CLI"

    local repo_path="$WORKSPACE/dit"

    # Remove any replace directives
    if ! $DRY_RUN; then
        cd "$repo_path"
        if grep -q "^replace" go.mod 2>/dev/null; then
            log_step "Removing replace directives from go.mod..."
            sed -i '/^replace/d' go.mod
        fi
    fi

    # Update all internal Go dependencies
    log_step "Updating CLI Go dependencies to v$VERSION..."
    for dep in remote-sdk-go s3-remote-go ssh-remote-go s3web-remote-go nop-remote-go dit-remote-go dit-client-go; do
        update_go_dep "$repo_path" "github.com/ditdotdev/$dep" "v$VERSION"
    done

    # Verify dependency alignment
    if ! $DRY_RUN; then
        cd "$repo_path"
        log_step "Verifying dependency alignment..."
        local sdk_versions
        sdk_versions=$(go mod graph | awk '$2 ~ /dit\/remote-sdk-go@/ {print $2}' | sort -u)
        local sdk_count
        sdk_count=$(echo "$sdk_versions" | wc -l)
        if [ "$sdk_count" -gt 1 ]; then
            log_error "Version conflict! Multiple remote-sdk-go versions detected:"
            echo "$sdk_versions"
            exit 1
        fi
        log_success "All providers use same remote-sdk-go version: $sdk_versions"
    fi

    # Update BATS test version expectations
    log_step "Updating BATS test version expectations..."
    local bats_file="$repo_path/tests/endtoend/remotes/dit/dit-workflow.bats"
    if [ -f "$bats_file" ] && ! $DRY_RUN; then
        sed -i "s/v${PREV_VERSION}/v${VERSION}/g" "$bats_file"
        log_success "Updated dit-workflow.bats versions"
    fi

    # Update DOWNLOAD_TEST_VERSION in both PROD and DEV sections of env.bash
    # This must happen BEFORE tagging so the tag includes these changes,
    # since dit-remote-server E2E checks out the CLI at this tag.
    local env_file="$repo_path/tests/endtoend/remotes/dit/env.bash"
    log_step "Updating DOWNLOAD_TEST_VERSION (PROD + DEV) to v$VERSION..."
    if ! $DRY_RUN; then
        # Guard against a moved/renamed env file: under `set -e` an unguarded
        # sed on a missing path aborts the whole release mid-phase.
        if [ -f "$env_file" ]; then
            sed -i "s/DOWNLOAD_TEST_VERSION:-v[0-9.]*}/DOWNLOAD_TEST_VERSION:-v$VERSION}/g" "$env_file"
            log_success "DOWNLOAD_TEST_VERSION updated in both PROD and DEV sections"
        else
            log_error "env file not found at $env_file — DOWNLOAD_TEST_VERSION not updated"
            exit 1
        fi
    fi

    # Commit dependency updates + env.bash
    if ! $DRY_RUN; then
        cd "$repo_path"
        if [ -n "$(git status --porcelain)" ]; then
            local files_to_add=(go.mod go.sum tests/endtoend/remotes/dit/env.bash)
            if [ -f "$bats_file" ] && git diff --name-only | grep -q "dit-workflow.bats"; then
                files_to_add+=(tests/endtoend/remotes/dit/dit-workflow.bats)
            fi
            commit_and_push "$repo_path" "Release v$VERSION: Update all dependencies" "${files_to_add[@]}"
        fi
    fi

    # Tag and release (triggers build + E2E + release creation)
    tag_and_push "$repo_path" "v$VERSION"

    # This is the longest workflow - builds 5 platforms + runs full E2E
    # (BATS suites + minikube + ZFS modprobe). The default WORKFLOW_TIMEOUT
    # (1800s) was the cap before v1.9.2; that release ran longer than 30
    # min and tripped the timeout even though the GH Actions run itself
    # succeeded. Double the wait for this one phase so the script doesn't
    # bail before the real workflow has finished.
    log_info "CLI release workflow typically takes 10-15 minutes (timeout doubled to ~60 min for headroom)..."
    WORKFLOW_TIMEOUT=$((WORKFLOW_TIMEOUT * 2)) wait_for_workflow "dit" "release.yml"
    verify_gh_release "dit" "v$VERSION"

    # Upload CLI binaries to production S3 + local dev MinIO
    local upload_script="$WORKSPACE/dit-remote-server/scripts/upload-release-to-minio.sh"
    if [ -f "$upload_script" ]; then
        log_step "Uploading CLI binaries to production S3..."
        if $DRY_RUN; then
            log_dry "bash $upload_script --version v$VERSION --minio-endpoint s3.us-west-2.amazonaws.com --minio-bucket $PROD_RELEASES_BUCKET --minio-use-ssl true"
        else
            bash "$upload_script" \
                --version "v$VERSION" \
                --minio-endpoint s3.us-west-2.amazonaws.com \
                --minio-bucket "$PROD_RELEASES_BUCKET" \
                --minio-use-ssl true
            log_success "CLI binaries uploaded to S3"

            # Verify S3 upload succeeded - remote-server E2E tests depend on these binaries
            log_step "Verifying S3 upload (required before Phase 7)..."
            verify_s3_artifact "$PROD_RELEASES_BUCKET" "v$VERSION/metadata.json"
            for platform in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows; do
                verify_s3_artifact "$PROD_RELEASES_BUCKET" "v$VERSION/$platform/"
            done
            log_success "All 5 platforms verified in S3 — safe to proceed to Phase 7"
        fi

        # Upload to local dev MinIO (only if reachable)
        log_step "Uploading CLI binaries to local dev MinIO ($DEV_MINIO_ENDPOINT)..."
        if $DRY_RUN; then
            log_dry "bash $upload_script --version v$VERSION --minio-endpoint $DEV_MINIO_ENDPOINT --minio-bucket $DEV_RELEASES_BUCKET --minio-use-ssl false"
        elif curl -sf -o /dev/null --max-time 3 "http://$DEV_MINIO_ENDPOINT/minio/health/live"; then
            bash "$upload_script" \
                --version "v$VERSION" \
                --minio-endpoint "$DEV_MINIO_ENDPOINT" \
                --minio-bucket "$DEV_RELEASES_BUCKET" \
                --minio-access-key minioadmin \
                --minio-secret-key minioadmin \
                --minio-use-ssl false
            log_success "CLI binaries uploaded to dev MinIO"
        else
            log_warn "Dev MinIO not reachable at $DEV_MINIO_ENDPOINT - skipping dev upload"
        fi
    else
        log_warn "Upload script not found at $upload_script - skipping S3/MinIO upload"
    fi

    save_phase_state 6
    log_success "Phase 6 complete: dit CLI released at v$VERSION"
}

# ============================================================================
# Phase 7: Remote Server
# ============================================================================

phase_remote_server() {
    log_phase 7 "dit-remote-server"

    local repo_path="$WORKSPACE/dit-remote-server"

    # Remove replace directives
    if ! $DRY_RUN; then
        cd "$repo_path"
        if grep -q "^replace" go.mod 2>/dev/null; then
            log_step "Removing replace directives from go.mod..."
            sed -i '/^replace/d' go.mod
            go mod tidy
        fi
    fi

    # Check if go.mod has dit deps to update
    if grep -q "github.com/ditdotdev/" "$repo_path/go.mod" 2>/dev/null; then
        update_go_dep "$repo_path" "github.com/ditdotdev/remote-sdk-go" "v$VERSION"
    fi

    # Commit if changes
    if ! $DRY_RUN; then
        cd "$repo_path"
        if [ -n "$(git status --porcelain)" ]; then
            commit_and_push "$repo_path" "Update dependencies for v$VERSION release" go.mod go.sum go.work.sum
        fi
    fi

    # Tag and release (triggers build 8 images → E2E → ECR publish)
    tag_and_push "$repo_path" "v$VERSION"

    log_info "Remote server workflow builds 8 Docker images + runs E2E tests. This takes 15-20 minutes..."
    wait_for_workflow "dit-remote-server" "release.yml"
    verify_gh_release "dit-remote-server" "v$VERSION"

    save_phase_state 7
    log_success "Phase 7 complete: dit-remote-server released at v$VERSION"
}

# ============================================================================
# Phase 8: ECS Deployment
# ============================================================================

phase_ecs_deploy() {
    log_phase 8 "AWS ECS Production Deployment"

    if $SKIP_ECS; then
        log_info "ECS deployment skipped (--skip-ecs)"
        save_phase_state 8
        return 0
    fi

    # 0. Apply Terraform changes BEFORE the bash deploy modifies live state
    # (issue #137). Catches drift in TF-owned infra (DNS, IAM, SSM, log
    # groups, IAM policies) every release so the source-vs-live gap stays
    # small. ECS service `task_definition` fields carry lifecycle
    # ignore_changes in the TF module, so apply won't try to roll back
    # task definitions registered by the bash deploy in step 3 below.
    # The Phase 0 pre-flight verified dit-remote-server's working
    # tree is clean, so what we apply matches what's checked in on master.
    local tf_dir="$WORKSPACE/dit-remote-server/deploy/terraform"
    if [ ! -d "$tf_dir" ]; then
        log_error "Terraform directory not found at $tf_dir"
        exit 1
    fi
    log_step "Applying Terraform (catches infrastructure drift before deploy)..."
    if $DRY_RUN; then
        log_dry "cd $tf_dir && terraform init -input=false"
        log_dry "cd $tf_dir && terraform apply -auto-approve -input=false"
    else
        if ! command -v terraform >/dev/null 2>&1; then
            log_error "terraform not found in PATH. Install from https://developer.hashicorp.com/terraform/downloads"
            exit 1
        fi
        cd "$tf_dir"
        terraform init -input=false
        terraform apply -auto-approve -input=false
        log_success "Terraform apply completed"
    fi

    # 1. Run Liquibase migrations against production database via EC2
    log_step "Running Liquibase migrations against production..."
    local liquibase_dir="$WORKSPACE/dit-remote-server/deploy/liquibase"
    local ssh_key="$WORKSPACE/dit-remote-server/dit-ecs-host.pem"
    local ec2_ip

    if [ ! -d "$liquibase_dir" ]; then
        log_error "Liquibase directory not found at $liquibase_dir"
        exit 1
    fi
    if [ ! -f "$ssh_key" ]; then
        log_error "SSH key not found at $ssh_key"
        exit 1
    fi

    if $DRY_RUN; then
        log_dry "Look up EC2 Elastic IP"
        log_dry "SCP $liquibase_dir to ec2-user@<EC2_IP>:/tmp/liquibase"
        log_dry "SSH: fetch DB credentials from SSM, run Liquibase Docker container"
        log_dry "SSH: clean up /tmp/liquibase"
    else
        # Find the production EC2 instance IP
        ec2_ip=$(aws ec2 describe-instances \
            --filters "Name=tag:Name,Values=dit-ecs-host-prod" "Name=instance-state-name,Values=running" \
            --query 'Reservations[0].Instances[0].PublicIpAddress' \
            --output text --region "$ECR_REGION" 2>/dev/null || echo "")

        if [ -z "$ec2_ip" ] || [ "$ec2_ip" = "None" ]; then
            log_error "Could not find running EC2 instance 'dit-ecs-host-prod'"
            exit 1
        fi
        log_info "EC2 instance IP: $ec2_ip"

        local ssh_opts="-i $ssh_key -o StrictHostKeyChecking=no -o ConnectTimeout=10"

        # Copy Liquibase changelogs to EC2
        log_info "Copying Liquibase changelogs to EC2..."
        ssh $ssh_opts ec2-user@"$ec2_ip" "rm -rf /tmp/liquibase && mkdir -p /tmp/liquibase/changelogs/v1.0.0"
        scp $ssh_opts -r "$liquibase_dir"/* ec2-user@"$ec2_ip":/tmp/liquibase/

        # Fetch DB credentials from SSM locally (EC2 doesn't have aws CLI)
        log_info "Fetching DB credentials from SSM Parameter Store..."
        local db_url db_password db_host
        db_url=$(aws ssm get-parameter \
            --name "/dit/prod/database/url" \
            --with-decryption --region "$ECR_REGION" \
            --query 'Parameter.Value' --output text)

        db_password=$(aws ssm get-parameter \
            --name "/dit/prod/db/password" \
            --with-decryption --region "$ECR_REGION" \
            --query 'Parameter.Value' --output text)

        if [ -z "$db_url" ] || [ -z "$db_password" ]; then
            log_error "Failed to fetch production DB credentials from SSM"
            exit 1
        fi

        # Extract host from postgres:// URL
        db_host=$(echo "$db_url" | sed -n 's|.*@\(.*\):5432/.*|\1|p')
        log_success "Retrieved DB credentials (host: $db_host)"

        # Run Liquibase on EC2, passing credentials via SSH
        log_info "Running Liquibase on EC2..."
        ssh $ssh_opts ec2-user@"$ec2_ip" bash -s "$db_host" "$db_password" <<'REMOTE_SCRIPT'
            set -euo pipefail
            DB_HOST="$1"
            DB_PASSWORD="$2"

            # Run Liquibase via Docker
            docker run --rm \
                -v /tmp/liquibase:/liquibase/changelog \
                liquibase/liquibase:4.20 \
                --changeLogFile=changelog-master.xml \
                --url="jdbc:postgresql://${DB_HOST}:5432/dit?sslmode=require" \
                --username="dit" \
                --password="$DB_PASSWORD" \
                update

            # Clean up
            rm -rf /tmp/liquibase
REMOTE_SCRIPT

        log_success "Liquibase migrations completed"
    fi

    # 2. Get ECR registry
    log_step "Getting ECR registry..."
    local ecr_registry
    if $DRY_RUN; then
        ecr_registry="XXXXXXXXX.dkr.ecr.us-west-2.amazonaws.com"
        log_dry "ECR registry: $ecr_registry"
    else
        ecr_registry=$(aws ecr describe-repositories --region "$ECR_REGION" \
            --repository-names ditdotdev/api-gateway \
            --query 'repositories[0].repositoryUri' --output text | cut -d'/' -f1)
        log_success "ECR registry: $ecr_registry"
    fi

    # 3. Retrieve image digests for all 8 services
    log_step "Retrieving v$VERSION image digests from ECR..."
    declare -A DIGESTS
    for service in "${ECR_SERVICES[@]}"; do
        if $DRY_RUN; then
            DIGESTS[$service]="sha256:dry-run-placeholder"
            log_dry "Digest for $service: sha256:dry-run-placeholder"
        else
            local digest
            digest=$(aws ecr describe-images \
                --repository-name "ditdotdev/$service" \
                --region "$ECR_REGION" \
                --image-ids "imageTag=v$VERSION" \
                --query 'imageDetails[0].imageDigest' \
                --output text 2>/dev/null || echo "NOT_FOUND")

            if [ "$digest" = "NOT_FOUND" ] || [ "$digest" = "None" ]; then
                log_error "Image not found in ECR: ditdotdev/$service:v$VERSION"
                exit 1
            fi
            DIGESTS[$service]=$digest
            log_info "  $service: $digest"
        fi
    done

    # 4. Update task definitions script with new digests
    local task_def_script="$WORKSPACE/dit-remote-server/update-task-definitions-with-digests.sh"
    if [ -f "$task_def_script" ]; then
        log_step "Updating task definition digests..."
        if ! $DRY_RUN; then
            # Build the new SERVICES array content
            local services_block="declare -A SERVICES=(\n"
            for service in "${ECR_SERVICES[@]}"; do
                services_block+="    [\"$service\"]=\"${DIGESTS[$service]}\"\n"
            done
            services_block+=")"

            # Replace the SERVICES block in the script
            # Use perl for multi-line replacement
            perl -i -0pe "s/declare -A SERVICES=\(.*?\)/$(echo -e "$services_block")/s" "$task_def_script"

            cd "$WORKSPACE/dit-remote-server"
            git add update-task-definitions-with-digests.sh
            git commit -m "Update task definition digests for v$VERSION"
            git push origin master
        fi

        # 5. Run the task definition update script
        log_step "Registering new ECS task definitions..."
        if $DRY_RUN; then
            log_dry "bash $task_def_script"
        else
            bash "$task_def_script"
        fi
    else
        log_warn "Task definition script not found at $task_def_script"
        log_warn "You may need to manually update ECS task definitions"
    fi

    # 6. Wait for deployment
    log_step "Waiting 120s for ECS deployment to complete..."
    if ! $DRY_RUN; then
        sleep 120
    fi

    # 7. Verify deployment
    log_step "Verifying ECS deployment status..."
    if ! $DRY_RUN; then
        local all_healthy=true
        for service_name in $(aws ecs list-services --cluster "$ECS_CLUSTER" --region "$ECR_REGION" \
            --query 'serviceArns[*]' --output text | xargs -n1 basename); do
            local svc_status
            svc_status=$(aws ecs describe-services \
                --cluster "$ECS_CLUSTER" \
                --services "$service_name" \
                --region "$ECR_REGION" \
                --query 'services[0].[status,runningCount,desiredCount,deployments[0].rolloutState]' \
                --output text 2>/dev/null || echo "UNKNOWN")
            log_info "  $service_name: $svc_status"
        done
    fi

    save_phase_state 8
    log_success "Phase 8 complete: ECS deployment finished"
}

# ============================================================================
# Phase 9: Post-release Validation
# ============================================================================

phase_validate() {
    log_phase 9 "Post-release Validation"

    echo ""
    echo "Release Summary for v$VERSION"
    echo "--------------------------------------------------------"
    printf "%-30s %-10s %-10s\n" "Component" "Tag" "Status"
    echo "--------------------------------------------------------"

    # Go components
    local go_repos=(remote-sdk-go "${GO_PROVIDERS[@]}" dit-client-go dit-docker-proxy dit dit-remote-server)
    for repo in "${go_repos[@]}"; do
        local status="..."
        if $DRY_RUN; then
            status="[dry-run]"
        elif gh release view "v$VERSION" --repo "$ORG/$repo" > /dev/null 2>&1; then
            status="[OK]"
        else
            status="[MISSING]"
        fi
        printf "%-30s %-10s %-10s\n" "$repo" "v$VERSION" "$status"
    done

    # Kotlin components (no v prefix)
    local kotlin_repos=(command-executor remote-sdk "${KOTLIN_PROVIDERS[@]}")
    for repo in "${kotlin_repos[@]}"; do
        local status="..."
        if $DRY_RUN; then
            status="[dry-run]"
        elif gh release view "$VERSION" --repo "$ORG/$repo" > /dev/null 2>&1; then
            status="[OK]"
        else
            # Kotlin repos may not create GitHub releases, check tags instead
            cd "$WORKSPACE/$repo" 2>/dev/null
            if git tag -l "$VERSION" | grep -q "$VERSION" 2>/dev/null; then
                status="[OK] (tag)"
            else
                status="[MISSING]"
            fi
        fi
        printf "%-30s %-10s %-10s\n" "$repo" "$VERSION" "$status"
    done

    # Docker images
    echo ""
    echo "Docker Images:"
    echo "  dit-server -> ditdotdev/dit:v$VERSION (DockerHub)"
    for svc in "${ECR_SERVICES[@]}"; do
        echo "  dit-remote-server -> ditdotdev/$svc:v$VERSION (ECR)"
    done

    echo ""
    echo "--------------------------------------------------------"

    # Verify dependency alignment in CLI
    if ! $DRY_RUN; then
        log_step "Verifying CLI dependency alignment..."
        cd "$WORKSPACE/dit"
        go mod graph | grep dit | grep remote-sdk-go || true
    fi

    # Production smoke test
    if ! $SKIP_ECS; then
        log_step "Running production smoke tests against $PROD_URL..."
        local smoke_failed=false

        if $DRY_RUN; then
            log_dry "curl -sfSo /dev/null -w '%{http_code}' $PROD_URL/health"
            log_dry "curl -sfSo /dev/null -w '%{http_code}' $PROD_URL/"
            log_dry "curl -sfSo /dev/null -w '%{http_code}' $PROD_URL/auth/login"
            log_dry "openssl s_client -connect dit.dev:443 (cert expiry check)"
        else
            # 1. Health check - API gateway
            local http_code
            http_code=$(curl -sfSo /dev/null -w '%{http_code}' --max-time 10 "$PROD_URL/health" 2>/dev/null || echo "000")
            if [ "$http_code" = "200" ]; then
                log_success "Health check: $PROD_URL/health -> $http_code"
            else
                log_error "Health check: $PROD_URL/health -> $http_code"
                smoke_failed=true
            fi

            # 2. Web UI
            http_code=$(curl -sfSo /dev/null -w '%{http_code}' --max-time 10 "$PROD_URL/" 2>/dev/null || echo "000")
            if [ "$http_code" = "200" ]; then
                log_success "Web UI: $PROD_URL/ -> $http_code"
            else
                log_error "Web UI: $PROD_URL/ -> $http_code"
                smoke_failed=true
            fi

            # 3. Auth endpoint (expect redirect 302/307 or 200)
            http_code=$(curl -so /dev/null -w '%{http_code}' --max-time 10 "$PROD_URL/auth/login" 2>/dev/null || echo "000")
            if [ "$http_code" = "200" ] || [ "$http_code" = "302" ] || [ "$http_code" = "307" ]; then
                log_success "Auth flow: $PROD_URL/auth/login -> $http_code"
            else
                log_error "Auth flow: $PROD_URL/auth/login -> $http_code"
                smoke_failed=true
            fi

            # 4. API version check
            local api_response
            api_response=$(curl -sf --max-time 10 "$PROD_URL/health" 2>/dev/null || echo "{}")
            if echo "$api_response" | grep -q '"healthy"'; then
                log_success "API response: healthy"
            else
                log_warn "API response: could not verify health payload"
            fi

            # 5. SSL certificate expiry check
            local cert_expiry
            cert_expiry=$(echo | openssl s_client -servername dit.dev -connect dit.dev:443 2>/dev/null | \
                openssl x509 -noout -enddate 2>/dev/null | sed 's/notAfter=//')
            if [ -n "$cert_expiry" ]; then
                local expiry_epoch
                expiry_epoch=$(date -d "$cert_expiry" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "$cert_expiry" +%s 2>/dev/null || echo "0")
                local now_epoch
                now_epoch=$(date +%s)
                local days_remaining=$(( (expiry_epoch - now_epoch) / 86400 ))
                if [ "$days_remaining" -gt 7 ]; then
                    log_success "SSL certificate: valid for $days_remaining days (expires $cert_expiry)"
                elif [ "$days_remaining" -gt 0 ]; then
                    log_warn "SSL certificate: expires in $days_remaining days! ($cert_expiry)"
                    smoke_failed=true
                else
                    log_error "SSL certificate: EXPIRED ($cert_expiry)"
                    smoke_failed=true
                fi
            else
                log_warn "SSL certificate: could not check expiry"
            fi

            if $smoke_failed; then
                log_error "Production smoke tests FAILED - investigate before announcing release"
                exit 1
            fi
            log_success "All production smoke tests passed"
        fi
    else
        log_info "Skipping production smoke tests (--skip-ecs)"
    fi

    # Run full BATS E2E test suite against production
    if ! $SKIP_ECS; then
        log_step "Running ENV=PROD E2E test suite against production..."
        if $DRY_RUN; then
            log_dry "cd $WORKSPACE/dit && ENV=PROD make e2e-server"
        else
            cd "$WORKSPACE/dit"
            if ENV=PROD make e2e-server; then
                log_success "Production E2E tests passed"
            else
                log_error "Production E2E tests FAILED - investigate before announcing release"
                exit 1
            fi
        fi
    else
        log_info "Skipping production E2E tests (--skip-ecs)"
    fi

    save_phase_state 9
    log_success "Release v$VERSION validation complete!"
    echo ""
    echo "Release v$VERSION is complete!"
}

# ============================================================================
# Argument Parsing
# ============================================================================

parse_args() {
    if [ $# -lt 1 ]; then
        echo "Usage: $0 VERSION [OPTIONS]"
        echo ""
        echo "VERSION: Target version without 'v' prefix (e.g., 1.6.1)"
        echo ""
        echo "Options:"
        echo "  --from-phase N   Resume from phase N"
        echo "  --phase N        Run only phase N"
        echo "  --dry-run        Show commands without executing"
        echo "  --skip-ecs       Skip ECS deployment (Phase 8)"
        echo "  --prev VERSION   Previous version (default: auto-detected)"
        echo "  --timeout SECS   Workflow timeout (default: 900)"
        exit 1
    fi

    VERSION="$1"
    V_VERSION="v$VERSION"
    shift

    while [ $# -gt 0 ]; do
        case "$1" in
            --from-phase)
                FROM_PHASE="$2"
                shift 2
                ;;
            --phase)
                SINGLE_PHASE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --skip-ecs)
                SKIP_ECS=true
                shift
                ;;
            --prev)
                PREV_VERSION="$2"
                shift 2
                ;;
            --timeout)
                WORKFLOW_TIMEOUT="$2"
                shift 2
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Auto-detect previous version if not specified
    if [ -z "$PREV_VERSION" ]; then
        PREV_VERSION=$(detect_prev_version)
    fi
    PREV_V_VERSION="v$PREV_VERSION"

    # Validate
    if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_error "Invalid version format: $VERSION (expected: X.Y.Z)"
        exit 1
    fi

    echo ""
    echo "Dit Release Automation"
    echo "------------------------------------------------"
    echo "  Target version:   v$VERSION (Go) / $VERSION (Kotlin)"
    echo "  Previous version: v$PREV_VERSION"
    echo "  Dry run:          $DRY_RUN"
    echo "  Skip ECS:         $SKIP_ECS"
    if [ "$SINGLE_PHASE" -ge 0 ]; then
        echo "  Single phase:     $SINGLE_PHASE"
    elif [ "$FROM_PHASE" -gt 0 ]; then
        echo "  Starting from:    Phase $FROM_PHASE"
    fi
    echo "  Workflow timeout:  ${WORKFLOW_TIMEOUT}s"
    echo "------------------------------------------------"
    echo ""
}

# ============================================================================
# Main
# ============================================================================

main() {
    parse_args "$@"

    local phases=(
        phase_preflight
        phase_go_foundation
        phase_kotlin_foundation
        phase_client
        phase_docker_proxy
        phase_server
        phase_cli
        phase_remote_server
        phase_ecs_deploy
        phase_validate
    )

    if [ "$SINGLE_PHASE" -ge 0 ]; then
        # Run single phase
        if [ "$SINGLE_PHASE" -ge ${#phases[@]} ]; then
            log_error "Phase $SINGLE_PHASE does not exist (max: $((${#phases[@]} - 1)))"
            exit 1
        fi
        ${phases[$SINGLE_PHASE]}
    else
        # Run phases from FROM_PHASE
        for i in "${!phases[@]}"; do
            if [ "$i" -ge "$FROM_PHASE" ]; then
                ${phases[$i]}
            else
                log_info "Skipping Phase $i (--from-phase $FROM_PHASE)"
            fi
        done
    fi
}

main "$@"
