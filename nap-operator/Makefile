# Image URL to use for building/pushing image targets
IMG ?= controller:latest

# Tool binaries are installed into ./bin (kept out of the project module)
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.16.4

# CRD options passed to controller-gen
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true"

.PHONY: all
all: build

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CRDs, RBAC, webhook manifests from kubebuilder markers.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate zz_generated.deepcopy.go files.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run controller from your host against the cluster in $KUBECONFIG.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build the controller docker image (multi-arch step expected on Apple Silicon).
	docker build -t ${IMG} .

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the cluster pointed to by $KUBECONFIG.
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the cluster pointed to by $KUBECONFIG.
	kubectl delete -f config/crd/bases/ --ignore-not-found

##@ Tools

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
