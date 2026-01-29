# Autopilot Security Sensor Makefile

# Variables
VERSION ?= 0.1.0
REGISTRY ?= gcr.io/invisible-sre-sandbox
AGENT_IMAGE := $(REGISTRY)/apss-agent
CONTROLLER_IMAGE := $(REGISTRY)/apss-controller
WEBHOOK_IMAGE := $(REGISTRY)/apss-webhook

# Go settings
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED := 0
LDFLAGS := -w -s
ifdef VERSION
LDFLAGS += -X github.com/invisible-tech/autopilot-security-sensor/internal/version.Version=$(VERSION)
endif

.PHONY: all build test clean docker-build docker-push deploy

all: build

## Build binaries
build: build-agent build-controller build-webhook

build-agent:
	@echo "Building agent..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags='$(LDFLAGS)' -o bin/apss-agent ./cmd/agent

build-controller:
	@echo "Building controller..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags='$(LDFLAGS)' -o bin/apss-controller ./cmd/controller

build-webhook:
	@echo "Building webhook..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags='$(LDFLAGS)' -o bin/apss-webhook ./cmd/webhook

## Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

## Build Docker images
docker-build: docker-build-agent docker-build-controller docker-build-webhook

docker-build-agent:
	@echo "Building agent image..."
	docker build -f build/Dockerfile.agent -t $(AGENT_IMAGE):$(VERSION) .
	docker tag $(AGENT_IMAGE):$(VERSION) $(AGENT_IMAGE):latest

docker-build-controller:
	@echo "Building controller image..."
	docker build -f build/Dockerfile.controller -t $(CONTROLLER_IMAGE):$(VERSION) .
	docker tag $(CONTROLLER_IMAGE):$(VERSION) $(CONTROLLER_IMAGE):latest

docker-build-webhook:
	@echo "Building webhook image..."
	docker build -f build/Dockerfile.webhook -t $(WEBHOOK_IMAGE):$(VERSION) .
	docker tag $(WEBHOOK_IMAGE):$(VERSION) $(WEBHOOK_IMAGE):latest

## Push Docker images
docker-push: docker-push-agent docker-push-controller docker-push-webhook

docker-push-agent:
	docker push $(AGENT_IMAGE):$(VERSION)
	docker push $(AGENT_IMAGE):latest

docker-push-controller:
	docker push $(CONTROLLER_IMAGE):$(VERSION)
	docker push $(CONTROLLER_IMAGE):latest

docker-push-webhook:
	docker push $(WEBHOOK_IMAGE):$(VERSION)
	docker push $(WEBHOOK_IMAGE):latest

## Deploy to cluster
deploy:
	@echo "Deploying to cluster..."
	@kubectl create namespace apss-system --dry-run=client -o yaml | kubectl apply -f - || true
	helm upgrade --install apss ./deploy/helm \
		--namespace apss-system \
		--set agent.image.tag=$(VERSION) \
		--set controller.image.tag=$(VERSION) \
		--set webhook.image.tag=$(VERSION)

## Deploy to specific cluster (sre-onboarding)
deploy-sre-onboarding:
	@echo "Deploying to sre-onboarding cluster..."
	kubectl config use-context gke_invisible-sre-sandbox_us-west1_sre-771-staging
	$(MAKE) deploy

## Uninstall from cluster
uninstall:
	helm uninstall apss --namespace apss-system

## Generate protobuf code
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/api/v1/events.proto

## Lint code
lint:
	golangci-lint run ./...

## Format code
fmt:
	go fmt ./...
	goimports -w .

## Show help
help:
	@echo "APSS - Autopilot Security Sensor"
	@echo ""
	@echo "Targets:"
	@echo "  build              - Build all binaries"
	@echo "  test               - Run tests"
	@echo "  docker-build       - Build Docker images"
	@echo "  docker-push        - Push Docker images to registry"
	@echo "  deploy             - Deploy to current cluster"
	@echo "  deploy-sre-onboarding - Deploy to sre-onboarding cluster"
	@echo "  uninstall          - Uninstall from cluster"
	@echo "  clean              - Clean build artifacts"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  REGISTRY=$(REGISTRY)"
