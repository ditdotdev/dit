# d3 CLI End-to-End Test Coverage

This document tracks E2E test coverage for every d3 CLI command and user path.

## Coverage Legend

- **Covered** - command is exercised in at least one BATS test
- **Partial** - some flags/paths tested, others missing
- **Not Covered** - no test coverage exists

## Command Coverage Matrix

### Core Repository Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `d3 install` | Covered | `infrastructure/install.bats` | |
| `d3 uninstall` | Covered | `infrastructure/uninstall.bats` | `--force`, `--remove-images` tested |
| `d3 upgrade` | Covered | `infrastructure/upgrade.bats` | Idempotent upgrade tested |
| `d3 ls` | Covered | `getting-started/getting-started.bats`, `remotes/datadatdat/datadatdat-workflow.bats` | |
| `d3 run` | Covered | `getting-started/getting-started.bats`, `context/docker/docker-tests.bats` | `-e`, `-P`, `-n` flags tested |
| `d3 start` | Covered | `container-lifecycle/container-lifecycle.bats` | Start previously stopped container |
| `d3 stop` | Covered | `container-lifecycle/container-lifecycle.bats` | Stop running container |
| `d3 status` | Covered | `container-lifecycle/container-lifecycle.bats`, `remotes/datadatdat/datadatdat-workflow.bats` | Running and stopped states |
| `d3 rm` | Covered | `getting-started/getting-started.bats`, `remotes/ssh/ssh-workflow.bats` | `-f` flag tested |
| `d3 commit` | Covered | `getting-started/getting-started.bats`, `remotes/s3/s3-workflow.bats` | `-m`, `-t` flags tested |
| `d3 checkout` | Covered | `getting-started/getting-started.bats`, `tags/tag-management.bats` | `--commit` and `--tags` tested |
| `d3 log` | Covered | `remotes/ssh/ssh-workflow.bats`, `tags/tag-management.bats` | `--tags` filter tested |
| `d3 clone` | Covered | `getting-started/getting-started.bats`, `remotes/datadatdat/clone-commit-workflow.bats` | `-c`, `-n`, `-P`, `-t` tested |
| `d3 cp` | Covered | `data-import/data-import.bats` | `-s`, `-d` flags tested |
| `d3 migrate` | Covered | `data-import/data-import.bats` | Unmanaged container migration |
| `d3 tag` | Covered | `tags/tag-management.bats` | Standalone `d3 tag` with `-c`, `-t` |
| `d3 delete` | Covered | `remotes/ssh/ssh-workflow.bats`, `tags/tag-management.bats` | Full delete and `--tags` removal |

### Remote Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `d3 push` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats`, `remotes/datadatdat/datadatdat-workflow.bats` | `-c`, `-r`, `-t` tested |
| `d3 push --update-only` | Covered | `push-pull/push-pull-options.bats` | Tag-only sync via S3 |
| `d3 pull` | Covered | `remotes/ssh/ssh-workflow.bats`, `remotes/datadatdat/datadatdat-workflow.bats` | |
| `d3 pull --update-only` | Covered | `push-pull/push-pull-options.bats` | Tag-only sync via S3 |
| `d3 push --tags` (datadatdat) | Covered | `remotes/datadatdat/push-pull-tags-remote.bats` | Tag-filtered push on datadatdat remote |
| `d3 pull --tags` (datadatdat) | Covered | `remotes/datadatdat/push-pull-tags-remote.bats` | Tag-filtered pull on datadatdat remote |
| `d3 clone` | Covered | `getting-started/getting-started.bats`, `tags/clone-tags.bats` | S3Web, S3, datadatdat remotes |
| `d3 abort` | Covered | `remotes/datadatdat/abort-workflow.bats` | No-op abort and basic abort |
| `d3 remote add` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | S3, SSH, datadatdat remotes |
| `d3 remote ls` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | |
| `d3 remote log` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | |
| `d3 remote rm` | Covered | `remotes/ssh/ssh-workflow.bats` | |

### Context Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `d3 context install` | Covered | `multi-context/multi-context.bats` | Docker type |
| `d3 context uninstall` | Covered | `multi-context/multi-context.bats` | With `--force` |
| `d3 context default` | Covered | `multi-context/multi-context.bats`, `context/context-list.bats` | Get and set |
| `d3 context ls` | Covered | `context/context-list.bats` | List contexts with headers |

### Auth Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `d3 auth login` | Covered | `remotes/datadatdat/org-workflow.bats`, `remotes/datadatdat/auth-status.bats` | `--api-key`, `--server` |
| `d3 auth status` | Covered | `remotes/datadatdat/auth-status.bats` | Before/after login/logout |
| `d3 auth logout` | Covered | `remotes/datadatdat/org-workflow.bats`, `remotes/datadatdat/auth-status.bats` | |

### Organization Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `d3 org list` | Covered | `remotes/datadatdat/org-workflow.bats` | Full CRUD lifecycle |
| `d3 org ls` | Covered | `remotes/datadatdat/org-workflow.bats` | Alias verified |

### Error Handling

| Scenario | Test File |
|----------|-----------|
| Non-existent repo operations | `error-handling/error-handling.bats` |
| Push/pull without remote | `error-handling/error-handling.bats` |
| Duplicate container name | `error-handling/error-handling.bats` |
| Clone from invalid URI | `error-handling/error-handling.bats` |

## Test Suites

### `make e2e` (local functionality)

Requires: Docker, d3 binary, AWS credentials (for S3 tests), SSH key (for SSH tests)

| Target | BATS File |
|--------|-----------|
| `test-install` | `infrastructure/install.bats` |
| `test-getting-started` | `getting-started/getting-started.bats` |
| `test-tags` | `tags/clone-tags.bats` |
| `test-tag-management` | `tags/tag-management.bats` |
| `test-docker-context` | `context/docker/docker-tests.bats` |
| `test-container-lifecycle` | `container-lifecycle/container-lifecycle.bats` |
| `test-context-list` | `context/context-list.bats` |
| `test-data-import` | `data-import/data-import.bats` |
| `test-error-handling` | `error-handling/error-handling.bats` |
| `test-upgrade` | `infrastructure/upgrade.bats` |
| `test-s3-workflow` | `remotes/s3/s3-workflow.bats` |
| `test-push-pull-options` | `push-pull/push-pull-options.bats` |
| `test-ssh-workflow` | `remotes/ssh/ssh-workflow.bats` |
| `test-uninstall` | `infrastructure/uninstall.bats` |

### `make e2e-server` (remote server functionality)

Requires: Docker, d3 binary, datadatdat-remote-server running locally

| Target | BATS File |
|--------|-----------|
| `test-install` | `infrastructure/install.bats` |
| `test-datadatdat-workflow` | `remotes/datadatdat/datadatdat-workflow.bats` |
| `test-clone-commit-workflow` | `remotes/datadatdat/clone-commit-workflow.bats` |
| `test-auth-workflow` | `remotes/datadatdat/auth-workflow.bats` |
| `test-auth-status` | `remotes/datadatdat/auth-status.bats` |
| `test-org-workflow` | `remotes/datadatdat/org-workflow.bats` |
| `test-billing-workflow` | `remotes/datadatdat/billing-workflow.bats` |
| `test-abort-workflow` | `remotes/datadatdat/abort-workflow.bats` |
| `test-push-pull-tags-remote` | `remotes/datadatdat/push-pull-tags-remote.bats` |
| `test-uninstall` | `infrastructure/uninstall.bats` |

### Standalone Targets (not in e2e or e2e-server)

| Target | BATS File | Notes |
|--------|-----------|-------|
| `test-multi-context` | `multi-context/multi-context.bats` | Excluded pending CI diagnosis |
| `test-db-matrix` | `db-matrix/db-matrix.bats` | Excluded pending CI diagnosis |
| `test-stripe-integration` | `remotes/datadatdat/stripe-integration.bats` | Requires Stripe API keys |
| `test-datadatdat-workflow-prod` | `remotes/datadatdat/datadatdat-workflow-prod.bats` | Production environment |
| `test-auth-workflow-prod` | `remotes/datadatdat/auth-workflow-prod.bats` | Production environment |

## Flag Coverage Detail

### Flags Tested

| Flag | Command(s) | Test File |
|------|-----------|-----------|
| `-n / --name` | run, clone | getting-started, clone-commit-workflow |
| `-f / --force` | rm, uninstall | getting-started, uninstall |
| `-m / --message` | commit | getting-started, s3-workflow |
| `-t / --tags` | commit, push, log, checkout, tag, delete, clone | clone-tags, tag-management, push-pull-tags-remote |
| `-c / --commit` | checkout, push, tag, delete | getting-started, clone-commit-workflow, tag-management |
| `-e / --env` | run | docker-tests |
| `-P / --disable-port-mapping` | run | docker-tests |
| `-r / --remote` | push, remote add | s3-workflow |
| `-s / --source` | cp | data-import |
| `-d / --destination` | cp | data-import |
| `-u / --update-only` | push, pull | push-pull-options |
| `-p / --parameters` | remote add, clone | ssh-workflow (keyFile param) |
| `--remove-images` | uninstall | uninstall |
| `--api-key` | auth login | auth-status, org-workflow |
| `--server` | auth login/logout/status, org list | auth-status, org-workflow |
| `--context` | (global) | multi-context |
| `--registry` | install | Not tested (infrastructure-specific) |
| `--privileged` | run | Not tested (requires privileged Docker) |
| `--path` | upgrade | Not tested (requires version mismatch) |

## Provider Coverage

| Provider | Push | Pull | Clone | Remote Add/Ls/Log/Rm |
|----------|------|------|-------|---------------------|
| S3 | s3-workflow, clone-tags, push-pull-options | - | - | s3-workflow |
| SSH | ssh-workflow | ssh-workflow | - | ssh-workflow |
| S3Web | - | - | getting-started | - |
| datadatdat | datadatdat-workflow | datadatdat-workflow | clone-commit-workflow | datadatdat-workflow |
| NOP | Not tested | Not tested | Not tested | Not tested |
