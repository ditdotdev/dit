# dit CLI End-to-End Test Coverage

This document tracks E2E test coverage for every dit CLI command and user path.

## Coverage Legend

- **Covered** - command is exercised in at least one BATS test
- **Partial** - some flags/paths tested, others missing
- **Not Covered** - no test coverage exists

## Command Coverage Matrix

### Core Repository Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `dit install` | Covered | `infrastructure/install.bats` | |
| `dit uninstall` | Covered | `infrastructure/uninstall.bats` | `--force`, `--remove-images` tested |
| `dit upgrade` | Covered | `infrastructure/upgrade.bats` | Idempotent upgrade tested |
| `dit ls` | Covered | `getting-started/getting-started.bats`, `remotes/dit/dit-workflow.bats` | |
| `dit run` | Covered | `getting-started/getting-started.bats`, `context/docker/docker-tests.bats` | `-e`, `-P`, `-n` flags tested |
| `dit start` | Covered | `container-lifecycle/container-lifecycle.bats` | Start previously stopped container |
| `dit stop` | Covered | `container-lifecycle/container-lifecycle.bats` | Stop running container |
| `dit status` | Covered | `container-lifecycle/container-lifecycle.bats`, `remotes/dit/dit-workflow.bats` | Running and stopped states |
| `dit rm` | Covered | `getting-started/getting-started.bats`, `remotes/ssh/ssh-workflow.bats` | `-f` flag tested |
| `dit commit` | Covered | `getting-started/getting-started.bats`, `remotes/s3/s3-workflow.bats`, `context/kubernetes/kubernetes-tests.bats` | `-m`, `-t` flags tested; on kubernetes the VolumeSnapshot must be ReadyToUse the moment `dit commit` returns (dit-server#313) |
| `dit checkout` | Covered | `getting-started/getting-started.bats`, `tags/tag-management.bats`, `context/kubernetes/kubernetes-tests.bats` | `--commit` and `--tags` tested; kubernetes checkout drops a row written right after `dit commit` returned (dit-server#313) |
| `dit log` | Covered | `remotes/ssh/ssh-workflow.bats`, `tags/tag-management.bats` | `--tags` filter tested |
| `dit clone` | Covered | `getting-started/getting-started.bats`, `remotes/dit/clone-commit-workflow.bats` | `-c`, `-n`, `-P`, `-t` tested |
| `dit cp` | Covered | `data-import/data-import.bats`, `db-matrix/db-matrix.bats` | `-s`, `-d` flags tested; the db-matrix suite copies a marker into every declared volume of every matrix image and reads it back after commit and checkout |
| `dit migrate` | Covered | `data-import/data-import.bats`, `db-matrix/db-matrix.bats` | Unmanaged PostgreSQL container with the Pagila schema; asserts the database survives migration, and that a failed copy rolls the repository back. The db-matrix suite migrates an unmanaged container of every matrix image and verifies a marker planted in each declared volume survives, plus one object created through the engine's own client (see below) |

#### Database matrix data verification (`db-matrix/db-matrix.bats`, #262)

For every matrix image with a declared volume, the suite creates one object through the engine's own client inside the container (one row, document, key, node, index entry or bucket object) right after `dit run`, and reads it back through the same client after `dit checkout`, after `dit clone` of the pushed commit, and after `dit migrate` of an unmanaged container. Each read waits for the engine to accept a query first, so "Up" alone is not treated as healthy. The per-engine readiness, write and verify commands are the `db_ready`, `db_write` and `db_verify` tables in `db-matrix/db-matrix-engines.bash`; `db-matrix/check-engines.sh <image>...` runs them against a plain container (write, restart, verify) without a dit install. Images whose image ships no client and no HTTP API reachable from inside the container keep the marker-only coverage and log that fact: `dynamodb-local`, `nats`, `weaviate`, `couchbase`, `tigergraph`, and `kafka` (no declared volume, so nothing to carry).
| `dit tag` | Covered | `tags/tag-management.bats` | Standalone `dit tag` with `-c`, `-t` |
| `dit delete` | Covered | `remotes/ssh/ssh-workflow.bats`, `tags/tag-management.bats` | Full delete and `--tags` removal |

### Remote Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `dit push` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats`, `remotes/dit/dit-workflow.bats` | `-c`, `-r`, `-t` tested |
| `dit push --update-only` | Covered | `push-pull/push-pull-options.bats` | Tag-only sync via S3 |
| `dit pull` | Covered | `remotes/ssh/ssh-workflow.bats`, `remotes/dit/dit-workflow.bats` | |
| `dit pull --update-only` | Covered | `push-pull/push-pull-options.bats` | Tag-only sync via S3 |
| `dit push --tags` (dit) | Covered | `remotes/dit/push-pull-tags-remote.bats` | Tag-filtered push on dit remote |
| `dit pull --tags` (dit) | Covered | `remotes/dit/push-pull-tags-remote.bats` | Tag-filtered pull on dit remote |
| `dit clone` | Covered | `getting-started/getting-started.bats`, `tags/clone-tags.bats` | S3Web, S3, dit remotes |
| `dit abort` | Covered | `remotes/dit/abort-workflow.bats` | No-op abort and basic abort |
| `dit remote add` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | S3, SSH, dit remotes |
| `dit remote ls` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | |
| `dit remote log` | Covered | `remotes/s3/s3-workflow.bats`, `remotes/ssh/ssh-workflow.bats` | |
| `dit remote rm` | Covered | `remotes/ssh/ssh-workflow.bats` | |

### Context Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `dit context install` | Covered | `multi-context/multi-context.bats`, `context/context-lifecycle.bats`, `context/kubernetes/kubernetes-context-edge.bats` | Docker + kubernetes types; name defaulting to type, duplicate-name collision, unknown type, invalid `-p`, stale-server reinstall cleanup (#214) |
| `dit context uninstall` | Covered | `multi-context/multi-context.bats`, `context/context-lifecycle.bats` | With `--force`; nonexistent-context error; targets the named context only |
| `dit context default` | Covered | `multi-context/multi-context.bats`, `context/context-list.bats`, `context/context-lifecycle.bats` | Get and set; nonexistent name fails without losing the current default |
| `dit context ls` | Covered | `context/context-list.bats`, `context/context-lifecycle.bats` | List contexts with headers; default marker |

### Auth Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `dit auth login` | Covered | `remotes/dit/org-workflow.bats`, `remotes/dit/auth-status.bats` | `--api-key`, `--server` |
| `dit auth status` | Covered | `remotes/dit/auth-status.bats` | Before/after login/logout |
| `dit auth logout` | Covered | `remotes/dit/org-workflow.bats`, `remotes/dit/auth-status.bats` | |

### Organization Commands

| Command | Status | Test File(s) | Notes |
|---------|--------|-------------|-------|
| `dit org list` | Covered | `remotes/dit/org-workflow.bats` | Full CRUD lifecycle |
| `dit org ls` | Covered | `remotes/dit/org-workflow.bats` | Alias verified |

### Error Handling

| Scenario | Test File |
|----------|-----------|
| Non-existent repo operations | `error-handling/error-handling.bats` |
| Push/pull without remote | `error-handling/error-handling.bats` |
| Duplicate container name | `error-handling/error-handling.bats` |
| Clone from invalid URI | `error-handling/error-handling.bats` |

## Test Suites

### `make e2e` (local functionality)

Requires: Docker, dit binary, AWS credentials (for S3 tests), SSH key (for SSH tests)

| Target | BATS File |
|--------|-----------|
| `test-install` | `infrastructure/install.bats` |
| `test-getting-started` | `getting-started/getting-started.bats` |
| `test-tags` | `tags/clone-tags.bats` |
| `test-tag-management` | `tags/tag-management.bats` |
| `test-docker-context` | `context/docker/docker-tests.bats` |
| `test-container-lifecycle` | `container-lifecycle/container-lifecycle.bats` |
| `test-context-list` | `context/context-list.bats` |
| `test-context-lifecycle` | `context/context-lifecycle.bats` |
| `test-kubernetes` | `context/kubernetes/kubernetes-tests.bats`, `context/kubernetes/kubernetes-context-edge.bats` |
| `test-data-import` | `data-import/data-import.bats` |
| `test-error-handling` | `error-handling/error-handling.bats` |
| `test-upgrade` | `infrastructure/upgrade.bats` |
| `test-s3-workflow` | `remotes/s3/s3-workflow.bats` |
| `test-push-pull-options` | `push-pull/push-pull-options.bats` |
| `test-ssh-workflow` | `remotes/ssh/ssh-workflow.bats` |
| `test-image-pull` | `image-pull/image-pull-policy.bats` |
| `test-uninstall` | `infrastructure/uninstall.bats` |

### `make e2e-server` (remote server functionality)

The `e2e-server` target lives in the **dit-remote-server** repo's Makefile
(run it from a dit-remote-server checkout; `DIT_DIR` defaults to a sibling
checkout of this repo). The BATS suites below still live in this repo.

Requires: Docker, dit binary, dit-remote-server running locally

| Target | BATS File |
|--------|-----------|
| `test-install` | `infrastructure/install.bats` |
| `test-dit-workflow` | `remotes/dit/dit-workflow.bats` |
| `test-clone-commit-workflow` | `remotes/dit/clone-commit-workflow.bats` |
| `test-auth-workflow` | `remotes/dit/auth-workflow.bats` |
| `test-auth-status` | `remotes/dit/auth-status.bats` |
| `test-org-workflow` | `remotes/dit/org-workflow.bats` |
| `test-billing-workflow` | `remotes/dit/billing-workflow.bats` |
| `test-abort-workflow` | `remotes/dit/abort-workflow.bats` |
| `test-push-pull-tags-remote` | `remotes/dit/push-pull-tags-remote.bats` |
| `test-uninstall` | `infrastructure/uninstall.bats` |

### Standalone Targets (not in e2e or e2e-server)

| Target | BATS File | Notes |
|--------|-----------|-------|
| `test-multi-context` | `multi-context/multi-context.bats` | Excluded pending CI diagnosis |
| `test-db-matrix` | `db-matrix/db-matrix.bats` | Excluded pending CI diagnosis |
| `test-stripe-integration` | `remotes/dit/stripe-integration.bats` | Requires Stripe API keys |
| `test-dit-workflow-prod` | `remotes/dit/dit-workflow-prod.bats` | Production environment |
| `test-auth-workflow-prod` | `remotes/dit/auth-workflow-prod.bats` | Production environment |

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
| `--pull` | run, clone | image-pull-policy (missing/always/never; clone runs the committed digest, #240), getting-started |
| `--path` | upgrade | Not tested (requires version mismatch) |

## Provider Coverage

| Provider | Push | Pull | Clone | Remote Add/Ls/Log/Rm |
|----------|------|------|-------|---------------------|
| S3 | s3-workflow, clone-tags, push-pull-options | - | - | s3-workflow |
| SSH | ssh-workflow | ssh-workflow | - | ssh-workflow |
| S3Web | - | - | getting-started | - |
| dit | dit-workflow | dit-workflow | clone-commit-workflow | dit-workflow |
| NOP | Not tested | Not tested | Not tested | Not tested |
