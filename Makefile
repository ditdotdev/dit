DATADATDAT_TARGET := build/d3
DATADATDAT_BIN := /usr/local/bin/d3
RELEASE_DIR := release
OS := "macos-latest"

# Version injection for build-time version setting
VERSION ?= dev
LDFLAGS := -ldflags "-X datadatdat/internal/app.DatadatdatVersion=$(VERSION)"

.PHONY: build release darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows clean

clean:
	rm -rf $(RELEASE_DIR)
	rm -rf build
	go clean -cache -modcache -testcache
	@echo "Cleaned all build artifacts and caches"

windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/windows/d3.exe cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/windows && zip datadatdat-cli-$(VERSION)-windows_amd64.zip d3.exe

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-amd64/d3 cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/linux-amd64 && tar -cvf datadatdat-cli-$(VERSION)-linux_amd64.tar d3

linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-arm64/d3 cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/linux-arm64 && tar -cvf datadatdat-cli-$(VERSION)-linux_arm64.tar d3

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-amd64/d3 cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/darwin-amd64 && zip datadatdat-cli-$(VERSION)-darwin_amd64.zip d3

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-arm64/d3 cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/darwin-arm64 && zip datadatdat-cli-$(VERSION)-darwin_arm64.zip d3

release: darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows
	@echo "Copying release binaries to root directory..."
	cp $(RELEASE_DIR)/windows/d3.exe d3.exe
	cp $(RELEASE_DIR)/linux-amd64/d3 d3-linux
	cp $(RELEASE_DIR)/darwin-arm64/d3 d3-mac
	@echo "Release complete! Binaries copied to root directory."

build:
	@echo "Building datadatdat with version $(VERSION)..."
	go build $(LDFLAGS) -o $(DATADATDAT_TARGET) cmd/datadatdat/datadatdat.go
	@echo "Build complete: $(DATADATDAT_TARGET)"

link:
	ln -s $(DATADATDAT_TARGET) $(DATADATDAT_BIN)

unlink:
	rm  $(DATADATDAT_BIN)

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

test-datadatdat-workflow:
	bats tests/endtoend/remotes/datadatdat/datadatdat-workflow.bats

test-datadatdat-workflow-prod:
	bats tests/endtoend/remotes/datadatdat/datadatdat-workflow-prod.bats

test-auth-workflow:
	bats tests/endtoend/remotes/datadatdat/auth-workflow.bats

test-org-workflow:
	bats tests/endtoend/remotes/datadatdat/org-workflow.bats

test-auth-workflow-prod:
	bats tests/endtoend/remotes/datadatdat/auth-workflow-prod.bats

test-multi-context:
	bats tests/endtoend/multi-context/multi-context.bats

# TODO: diagnose test-multi-context test-db-matrix in gh actions and readd to e2e
e2e: test-install test-getting-started test-tags test-docker-context test-s3-workflow test-ssh-workflow test-uninstall

e2e-server: test-install test-datadatdat-workflow test-auth-workflow test-org-workflow test-uninstall