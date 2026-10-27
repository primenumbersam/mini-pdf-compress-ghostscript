.PHONY: all build-linux build-windows clean

DIST_DIR := dist

all: clean build-linux build-windows

build-linux:
	@mkdir -p $(DIST_DIR)
	@echo "[*] Building Linux amd64 binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/mini-pdf-compress-linux-amd64 .
	@chmod +x $(DIST_DIR)/mini-pdf-compress-linux-amd64
	@echo "[+] Linux binary created: $(DIST_DIR)/mini-pdf-compress-linux-amd64"

build-windows:
	@mkdir -p $(DIST_DIR)
	@echo "[*] Building Windows amd64 executable..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H=windowsgui" -o $(DIST_DIR)/mini-pdf-compress-windows-amd64.exe .
	@echo "[+] Windows binary created: $(DIST_DIR)/mini-pdf-compress-windows-amd64.exe"

clean:
	@rm -rf $(DIST_DIR)
	@echo "[*] Cleaned $(DIST_DIR) directory"
