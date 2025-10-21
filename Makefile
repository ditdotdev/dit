VEXRUN_FILE := $(PWD)/utils/vexrun.jar
VEXRUN := java -jar $(VEXRUN_FILE)
DATADATDAT_TARGET := $(PWD)/build/d3
DATADATDAT_BIN := /usr/local/bin/d3
RELEASE_DIR := $(PWD)/release
OS := "macos-latest"

# Version injection for build-time version setting
VERSION ?= dev
LDFLAGS := -ldflags "-X datadatdat/internal/app.DatadatdatVersion=$(VERSION)"

.PHONY: build release darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows clean

clean:
	rm -rf $(RELEASE_DIR)
	rm -rf $(PWD)/build
	go clean -cache -modcache -testcache
	@echo "Cleaned all build artifacts and caches"

windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/windows/d3.exe $(PWD)/cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/windows && zip datadatdat-cli-$(VERSION)-windows_amd64.zip d3.exe

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-amd64/d3 $(PWD)/cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/linux-amd64 && tar -cvf datadatdat-cli-$(VERSION)-linux_amd64.tar d3

linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/linux-arm64/d3 $(PWD)/cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/linux-arm64 && tar -cvf datadatdat-cli-$(VERSION)-linux_arm64.tar d3

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-amd64/d3 $(PWD)/cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/darwin-amd64 && zip datadatdat-cli-$(VERSION)-darwin_amd64.zip d3

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/darwin-arm64/d3 $(PWD)/cmd/datadatdat/datadatdat.go
	cd $(RELEASE_DIR)/darwin-arm64 && zip datadatdat-cli-$(VERSION)-darwin_arm64.zip d3

release: darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows
	@echo "Copying release binaries to root directory..."
	cp $(RELEASE_DIR)/windows/d3.exe d3.exe
	cp $(RELEASE_DIR)/linux-amd64/d3 d3-linux
	@echo "Release complete! Binaries copied to root directory."

build:
	@echo "Building datadatdat with version $(VERSION)..."
	go build $(LDFLAGS) -o $(DATADATDAT_TARGET) $(PWD)/cmd/datadatdat/datadatdat.go
	@echo "Build complete: $(DATADATDAT_TARGET)"

link:
	ln -s $(DATADATDAT_TARGET) $(DATADATDAT_BIN)

unlink:
	rm  $(DATADATDAT_BIN)

test-setup:
	curl -Ls https://github.com/datadatdat/vexrun/releases/download/v0.0.5/vexrun-0.0.5.jar -z $(VEXRUN_FILE) -o $(VEXRUN_FILE)

test-install:
	$(VEXRUN) -f $(PWD)/tests/endtoend/infrastructure/Install.yml

test-uninstall:
	$(VEXRUN) -f $(PWD)/tests/endtoend/infrastructure/Uninstall.yml

test-getting-started:
	$(VEXRUN) -d $(PWD)/tests/endtoend/getting-started

test-tags:
	$(VEXRUN) -d $(PWD)/tests/endtoend/tags

test-db-matrix:
	$(VEXRUN) -f $(PWD)/tests/endtoend/db-matrix/databases.yml

test-docker-context:
	docker pull datadatdat/nginx-test:latest
	docker tag datadatdat/nginx-test:latest nginx-test
	$(VEXRUN) -d $(PWD)/tests/endtoend/context/docker

test-s3-workflow:
	$(VEXRUN) -f $(PWD)/tests/endtoend/remotes/s3/s3WorkflowTests.yml

test-ssh-workflow:
	$(VEXRUN) -f $(PWD)/tests/endtoend/remotes/ssh/sshWorkflowTests.yml

test-datadatdat-workflow:
	$(VEXRUN) -f $(PWD)/tests/endtoend/remotes/datadatdat/datadatdatWorkflowTests.yml

test-multi-context:
	$(VEXRUN) -d $(PWD)/tests/endtoend/multi-context

#e2e: test-setup test-install test-getting-started test-tags test-docker-context test-s3-workflow test-ssh-workflow test-datadatdat-workflow test-uninstall
e2e: test-setup test-install test-getting-started test-tags test-docker-context test-s3-workflow test-ssh-workflow test-uninstall