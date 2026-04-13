.PHONY: build clean

APP_NAME := gorpm
DIST_DIR := dist
LDFLAGS := -s -w

build:
	@mkdir -p $(DIST_DIR)
	@echo "Building for Apple Silicon (macOS/arm64)..."
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-macos-arm64 .
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-windows-x64.exe .
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-linux-x64 .
	@echo "Building for Linux (386)..."
	GOOS=linux GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-linux-x86 .

clean:
	rm -rf $(DIST_DIR)
