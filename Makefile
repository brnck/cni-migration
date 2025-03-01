help:  ## display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build_linux: ## build cni-migration
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o cni-migration ./cmd/.

build_mac: ## build cni-migration
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -o cni-migration-mac ./cmd/.

all: build_linux build_mac # build all targets
