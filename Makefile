PROJECT := arangodb-exporter
ifndef SCRIPTDIR
	SCRIPTDIR := $(shell pwd)
endif
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)
VERSION := $(shell cat $(ROOTDIR)/VERSION)
VERSION_MAJOR_MINOR_PATCH := $(shell echo $(VERSION) | cut -f 1 -d '+')
VERSION_MAJOR_MINOR := $(shell echo $(VERSION_MAJOR_MINOR_PATCH) | cut -f 1,2 -d '.')
VERSION_MAJOR := $(shell echo $(VERSION_MAJOR_MINOR) | cut -f 1 -d '.')
COMMIT := $(shell git rev-parse --short HEAD)
MAKEFILE := $(ROOTDIR)/Makefile
UBI := registry.access.redhat.com/ubi8/ubi-minimal:8.0

ifndef NODOCKER
	DOCKERCLI := $(shell which docker)
	GOBUILDLINKTARGET := ../../../..
else
	DOCKERCLI := 
	GOBUILDLINKTARGET := $(ROOTDIR)
endif

ifndef BUILDDIR
	BUILDDIR := $(ROOTDIR)
endif
GOBUILDDIR := $(BUILDDIR)/.gobuild
SRCDIR := $(SCRIPTDIR)
BINDIR := $(BUILDDIR)/bin
VENDORDIR := $(SCRIPTDIR)/vendor

ORGPATH := github.com/arangodb-helper
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.10.1-alpine

ifndef DOCKERTAG 
	DOCKERTAG := dev
endif
DOCKERIMAGE := $(DOCKERNAMESPACE)/arangodb-exporter:$(DOCKERTAG)

RELEASE := $(SRCDIR)/bin/release$(shell go env GOEXE)
GHRELEASE := $(SRCDIR)/bin/github-release$(shell go env GOEXE)
GOX := $(SRCDIR)/bin/gox$(shell go env GOEXE)
GOIMPORTS := $(SRCDIR)/bin/fmt$(shell go env GOEXE)
ADDLICENSE := $(SRCDIR)/bin/addlicense$(shell go env GOEXE)

# Magical rubbish to teach make what commas and spaces are.
EMPTY :=
SPACE := $(EMPTY) $(EMPTY)
COMMA := $(EMPTY),$(EMPTY)

ARCHS:=amd64 arm arm64
PLATFORMS:=$(subst $(SPACE),$(COMMA),$(foreach arch,$(ARCHS),linux/$(arch)))

GO_IGNORED:=vendor .gobuild

GO_SOURCES_QUERY := find $(SRCDIR) -name '*.go' -type f $(foreach IGNORED,$(GO_IGNORED),-not -path '$(SRCDIR)/$(IGNORED)/*' )
GO_SOURCES := $(shell $(GO_SOURCES_QUERY) | sort | uniq)
GO_SOURCES_PACKAGES := $(shell $(GO_SOURCES_QUERY) -exec dirname {} \; | sort | uniq)

SOURCES := $(shell find $(SRCDIR) -name '*.go' -not -path './test/*')

.PHONY: all clean build docker

all: build

clean:
	rm -Rf $(BINDIR) $(ROOTDIR)/arangodb-exporter

build: $(GOX)
	CGO_ENABLED=0 $(GOX) \
		-os="darwin linux windows" \
		-arch="$(ARCHS)" \
		-osarch="!darwin/arm !darwin/arm64" \
		-ldflags="-X main.projectVersion=${VERSION} -X main.projectBuild=${COMMIT}" \
		-output="bin/{{.OS}}/{{.Arch}}/arangodb-exporter" \
		-tags="netgo" \
		github.com/arangodb-helper/arangodb-exporter

.PHONY: check-vars
check-vars:
ifndef DOCKERNAMESPACE
	@echo "DOCKERNAMESPACE must be set"
	@exit 1
endif
	@echo "Using docker namespace: $(DOCKERNAMESPACE)"

$(GOBUILDDIR):
	# pass

.PHONY: run-tests
run-tests: 
	go test $(REPOPATH)

docker-ubi-base: check-vars
	docker build --no-cache -t $(DOCKERIMAGE)-base-image-ubi -f Dockerfile.ubi .

docker-ubi: docker-ubi-base build
ifndef LOCAL
	for arch in amd64; do \
		docker build --build-arg "GOARCH=$$arch" --build-arg "VERSION=$(VERSION_MAJOR_MINOR_PATCH)" -t $(DOCKERIMAGE)-ubi --build-arg "BASE_IMAGE=$(DOCKERIMAGE)-base-image-ubi" -f Dockerfile.scratch . ; \
		docker push $(DOCKERIMAGE)-ubi ; \
	done
else
	for arch in amd64; do \
		docker build --build-arg "GOARCH=$$arch" --build-arg "VERSION=$(VERSION_MAJOR_MINOR_PATCH)" -t $(DOCKERIMAGE)-ubi --build-arg "BASE_IMAGE=$(DOCKERIMAGE)-base-image-ubi" -f Dockerfile.scratch . ; \
	done
endif

ifndef IGNORE_UBI
docker: docker-ubi
endif

docker: check-vars build
ifndef LOCAL
	for arch in $(ARCHS); do \
		docker build --build-arg "GOARCH=$$arch" --build-arg "VERSION=$(VERSION_MAJOR_MINOR_PATCH)" -t $(DOCKERIMAGE)-$$arch -f Dockerfile.scratch . ; \
		docker push $(DOCKERIMAGE)-$$arch ; \
	done
	docker tag $(DOCKERIMAGE)-amd64 $(DOCKERIMAGE)
	docker push $(DOCKERIMAGE)
else
	for arch in $(ARCHS); do \
		docker build --build-arg "GOARCH=$$arch" --build-arg "VERSION=$(VERSION_MAJOR_MINOR_PATCH)" -t $(DOCKERIMAGE)-$$arch -f Dockerfile.scratch . ; \
	done
endif

$(RELEASE): $(GOBUILDDIR) $(SOURCES) $(GHRELEASE)
	go build -o $(RELEASE) $(REPOPATH)/tools/release

$(GHRELEASE): $(GOBUILDDIR)
	@go build -mod='' -o "$(GHRELEASE)" github.com/aktau/github-release

github-release: $(GHRELEASE)

$(GOX): 
	go build -o $(GOX) github.com/mitchellh/gox

release-patch: $(RELEASE) $(GHRELEASE)
	$(RELEASE) -type=patch 

release-minor: $(RELEASE) $(GHRELEASE)
	$(RELEASE) -type=minor

release-major: $(RELEASE) $(GHRELEASE)
	$(RELEASE) -type=major 

## LINT

GOLANGCI_ENABLED=deadcode gocyclo golint varcheck structcheck maligned errcheck \
                 ineffassign interfacer unconvert goconst \
                 megacheck

.PHONY: tools
tools: $(ADDLICENSE) $(GOIMPORTS) $(GHRELEASE)

$(ADDLICENSE):
	@go build -mod='' -o "$(ADDLICENSE)" github.com/google/addlicense

.PHONY: license
license: $(ADDLICENSE)
	@echo ">> Verify license of files"
	@$(ADDLICENSE) -f "./LICENSE.BOILERPLATE" $(GO_SOURCES)

.PHONY: license-verify
license-verify: $(ADDLICENSE)
	@echo ">> Ensuring license of files"
	@$(ADDLICENSE) -f "./LICENSE.BOILERPLATE" -check $(GO_SOURCES)

$(GOIMPORTS):
	@go build -mod='' -o "$(GOIMPORTS)" golang.org/x/tools/cmd/goimports

.PHONY: fmt
fmt: $(GOIMPORTS)
	@echo ">> Ensuring style of files"
	@$(GOIMPORTS) -w $(GO_SOURCES)

.PHONY: fmt-verify
fmt-verify: license-verify $(GOIMPORTS)
	@echo ">> Verify files style"
	@if [ X"$$($(GOIMPORTS) -l $(GO_SOURCES) | wc -l)" != X"0" ]; then echo ">> Style errors"; $(GOIMPORTS) -l $(GO_SOURCES); exit 1; fi

.PHONY: linter
linter: fmt
	@golangci-lint run --no-config --issues-exit-code=1 --deadline=30m --disable-all \
	                  $(foreach MODE,$(GOLANGCI_ENABLED),--enable $(MODE) ) \
	                  --exclude-use-default=false \
	                  $(GO_SOURCES_PACKAGES)

.PHONY: vendor
vendor:
	@go mod vendor