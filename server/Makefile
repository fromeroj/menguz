APP      := menguz-server
BIN      := bin/$(APP)
DIST     := ../design/app/dist
LDFLAGS  := -s -w
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build run test vet tidy frontend deploy backup restore clean

## build: compile the server binary (Linux AMD64 by default for deploy; native for local)
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -X main.version=$(VERSION)" -o $(BIN) .

## build-linux: cross-compile for the production VPS
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN)-linux-amd64 .

## run: build the frontend and run the server locally (mock SAE data)
run: frontend build
	./$(BIN) --port=8080 --static-dir=$(DIST)

## dev: run with hot reload (install air: go install github.com/cosmtrek/air@latest)
dev:
	air -- --port=8080 --static-dir=$(DIST)

## frontend: build the React storefront into design/app/dist
frontend:
	cd ../design/app && npm run build

test:
	go test ./...

vet:
	go vet ./...
	gofmt -l .

tidy:
	go mod tidy

## deploy: scp binary to VPS and restart the systemd service
deploy: build-linux
	scp $(BIN)-linux-amd64 $(VPS_USER)@$(VPS_HOST):/opt/menguz/$(APP)
	ssh $(VPS_USER)@$(VPS_HOST) "systemctl restart menguz"

## backup: rsync the data dir to the backup host
backup:
	rsync -avz --delete $(DATA_DIR)/ $(BACKUP_HOST):/backups/menguz/$$(date +%Y%m%d)/

## restore: restore the latest backup into ./data (server must be stopped)
restore:
	./scripts/restore.sh

clean:
	rm -rf bin/ data/
