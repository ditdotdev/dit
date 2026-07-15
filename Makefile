DIT_TARGET := build/dit
DIT_BIN := /usr/local/bin/dit
RELEASE_DIR := release
OS := "macos-latest"

# Environment: DEV (default) or PROD
ENV ?= DEV

# Version injection for build-time version setting
VERSION ?= dev
LDFLAGS := -ldflags "-X github.com/ditdotdev/dit/internal/app.DitVersion=$(VERSION)"

.PHONY: build release darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows clean coverage gen-docs k8s-csi-default

# Regenerate the Markdown CLI reference under docs/src/cli/cmd/ from the live
# Cobra command tree. CI runs the same target and then `git diff --exit-code`
# against that directory, so any CLI flag/help change must ship with regenerated
# docs or the PR fails.
gen-docs:
	go run ./cmd/gen-docs

# Runs go test ./... and prints both the raw % (what `go test -cover` reports)
# and the scored % the CI repo-health gate uses (filters generated + main
# packages — see scripts/local-coverage.sh for the regex, kept in sync with
# ditdotdev/.github/.github/workflows/repo-health.yml).
coverage:
	bash scripts/local-coverage.sh

clean:
	rm -rf $(RELEASE_DIR)
	rm -rf build
	go clean -cache -modcache -testcache
	@echo "Cleaned all build artifacts and caches"

windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/windows/dit.exe cmd/dit/dit.go
	cd $(RELEASE_DIR)/windows && zip dit-cli-$(VERSION)-windows_amd64.zip dit.exe

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-amd64/dit cmd/dit/dit.go
	cd $(RELEASE_DIR)/linux-amd64 && tar -cvf dit-cli-$(VERSION)-linux_amd64.tar dit

linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-arm64/dit cmd/dit/dit.go
	cd $(RELEASE_DIR)/linux-arm64 && tar -cvf dit-cli-$(VERSION)-linux_arm64.tar dit

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-amd64/dit cmd/dit/dit.go
	cd $(RELEASE_DIR)/darwin-amd64 && zip dit-cli-$(VERSION)-darwin_amd64.zip dit

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-arm64/dit cmd/dit/dit.go
	cd $(RELEASE_DIR)/darwin-arm64 && zip dit-cli-$(VERSION)-darwin_arm64.zip dit

release: darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows
	@echo "Copying release binaries to root directory..."
	cp $(RELEASE_DIR)/windows/dit.exe dit.exe
	cp $(RELEASE_DIR)/linux-amd64/dit dit-linux
	cp $(RELEASE_DIR)/darwin-arm64/dit dit-mac
	@echo "Release complete! Binaries copied to root directory."

build:
	@echo "Building dit with version $(VERSION)..."
	go build $(LDFLAGS) -o $(DIT_TARGET) cmd/dit/dit.go
	@echo "Build complete: $(DIT_TARGET)"

link:
	ln -s $(DIT_TARGET) $(DIT_BIN)

unlink:
	rm  $(DIT_BIN)

test-install:
	bats tests/endtoend/infrastructure/install.bats

test-uninstall:
	bats tests/endtoend/infrastructure/uninstall.bats

test-getting-started:
	bats tests/endtoend/getting-started/getting-started.bats

test-tags:
	bats tests/endtoend/tags/clone-tags.bats

test-db-matrix:
	bats tests/endtoend/db-matrix/db-matrix.bats

test-docker-context:
	bats tests/endtoend/context/docker/docker-tests.bats

test-s3-workflow:
	bats tests/endtoend/remotes/s3/s3-workflow.bats

test-ssh-workflow:
	bats tests/endtoend/remotes/ssh/ssh-workflow.bats

test-dit-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/dit-workflow.bats

test-auth-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/auth-workflow.bats

test-org-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/org-workflow.bats

test-clone-commit-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/clone-commit-workflow.bats

test-billing-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/billing-workflow.bats

test-stripe-integration:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/stripe-integration.bats

test-favicon:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/favicon.bats

test-multi-context:
	bats tests/endtoend/multi-context/multi-context.bats

test-container-lifecycle:
	bats tests/endtoend/container-lifecycle/container-lifecycle.bats

test-data-import:
	bats tests/endtoend/data-import/data-import.bats

test-tag-management:
	bats tests/endtoend/tags/tag-management.bats

test-upgrade:
	bats tests/endtoend/infrastructure/upgrade.bats

test-context-list:
	bats tests/endtoend/context/context-list.bats

test-context-lifecycle:
	bats tests/endtoend/context/context-lifecycle.bats

test-error-handling:
	bats tests/endtoend/error-handling/error-handling.bats

test-push-pull-options:
	bats tests/endtoend/push-pull/push-pull-options.bats

test-abort-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/abort-workflow.bats

test-auth-status:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/auth-status.bats

test-push-pull-tags-remote:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/push-pull-tags-remote.bats

test-fork-workflow:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/fork-workflow.bats

test-fork-cross-user:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/fork-cross-user.bats

test-whitelist-approval:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/whitelist-approval.bats

test-public-repo-permissions:
	ENV=$(ENV) bats tests/endtoend/remotes/dit/public-repo-permissions.bats

# Local-minikube helper: assert the CSI hostpath class as the cluster default
# (and demote the non-CSI `standard` class) so dit-provisioned volumes use the
# CSI driver and `dit commit` VolumeSnapshots can become ReadyToUse. minikube's
# `default-storageclass` addon re-asserts `standard` on every `minikube start`,
# so run this after each start. Idempotent; tolerates failures (|| true). CI
# already does this in release.yml / pull-request.yml.
k8s-csi-default:
	kubectl patch storageclass csi-hostpath-sc -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}' || true
	kubectl patch storageclass standard -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}' || true

# Kubernetes provider tests. Both targets self-skip when no k8s cluster is
# reachable (kubectl cluster-info), so they are safe to include in `e2e` /
# `e2e-server` on hosts without minikube — they just no-op rather than fail.
# Local prerequisite (minikube): run `make k8s-csi-default` first so volumes
# use the CSI driver, otherwise VolumeSnapshots fail with
# "snapshotting non-CSI volumes is not supported".
test-kubernetes:
	bats tests/endtoend/context/kubernetes/kubernetes-tests.bats
	bats tests/endtoend/context/kubernetes/kubernetes-context-edge.bats

test-kubernetes-remote:
	ENV=$(ENV) bats tests/endtoend/context/kubernetes/kubernetes-remote-tests.bats

# TODO: diagnose test-multi-context test-db-matrix in gh actions and readd to e2e
e2e: test-install test-getting-started test-tags test-tag-management test-docker-context test-container-lifecycle test-context-list test-context-lifecycle test-data-import test-error-handling test-s3-workflow test-push-pull-options test-ssh-workflow test-kubernetes test-upgrade test-uninstall

test-connect-drs-network:
	docker network connect dit-docker dit-docker-server 2>/dev/null || true

e2e-server: test-install test-connect-drs-network test-dit-workflow test-clone-commit-workflow test-auth-workflow test-whitelist-approval test-public-repo-permissions test-auth-status test-org-workflow test-billing-workflow test-stripe-integration test-favicon test-abort-workflow test-push-pull-tags-remote test-fork-workflow test-fork-cross-user test-kubernetes-remote test-uninstall