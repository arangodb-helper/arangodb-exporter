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

# Magical rubbish to teach make what commas and spaces are.
EMPTY :=
SPACE := $(EMPTY) $(EMPTY)
COMMA := $(EMPTY),$(EMPTY)

ARCHS:=amd64 arm arm64
PLATFORMS:=$(subst $(SPACE),$(COMMA),$(foreach arch,$(ARCHS),linux/$(arch)))

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

docker: check-vars build
	for arch in $(ARCHS); do \
		docker build --build-arg=GOARCH=$$arch -t $(DOCKERIMAGE)-$$arch . ;\
		docker push $(DOCKERIMAGE)-$$arch ;\
	done
	docker manifest create --amend $(DOCKERIMAGE) $(foreach arch,$(ARCHS),$(DOCKERIMAGE)-$(arch))
	docker manifest push $(DOCKERIMAGE)

$(RELEASE): $(GOBUILDDIR) $(SOURCES) $(GHRELEASE)
	go build -o $(RELEASE) $(REPOPATH)/tools/release

$(GHRELEASE): $(GOBUILDDIR) 
	go build -o $(GHRELEASE) github.com/aktau/github-release

$(GOX): 
	go build -o $(GOX) github.com/mitchellh/gox

release-patch: $(RELEASE)
	$(RELEASE) -type=patch 

release-minor: $(RELEASE)
	$(RELEASE) -type=minor

release-major: $(RELEASE)
	$(RELEASE) -type=major 

