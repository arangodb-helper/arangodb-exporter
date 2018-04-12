PROJECT := arangodb_exporter
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

ORGPATH := github.com/arangodb-helper
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.10.1-alpine

ifndef GOOS
	GOOS := linux
endif
ifndef GOARCH
	GOARCH := amd64
endif
ifeq ("$(GOOS)", "windows")
	GOEXE := .exe
endif

ifndef DOCKERNAMESPACE
	DOCKERNAMESPACE := arangodb
endif

BINNAME := arangodb_exporter$(GOEXE)
BIN := $(BINDIR)/$(GOOS)/$(GOARCH)/$(BINNAME)
RELEASE := $(GOBUILDDIR)/bin/release 
GHRELEASE := $(GOBUILDDIR)/bin/github-release 

SOURCES := $(shell find $(SRCDIR) -name '*.go' -not -path './test/*')

.PHONY: all clean deps docker build build-local

all: build

clean:
	rm -Rf $(BIN) $(BINDIR) $(GOBUILDDIR) $(ROOTDIR)/arangodb_exporter

local:
ifneq ("$(DOCKERCLI)", "")
	@${MAKE} -f $(MAKEFILE) -B GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) build-local
else
	@${MAKE} -f $(MAKEFILE) deps
	GOPATH=$(GOBUILDDIR) go build -o $(BUILDDIR)/arangodb $(REPOPATH)
endif

build: $(BIN)

build-local: build 
	@ln -sf $(BIN) $(ROOTDIR)/arangodb

binaries: $(GHRELEASE)
	@${MAKE} -f $(MAKEFILE) -B GOOS=linux GOARCH=amd64 build
	@${MAKE} -f $(MAKEFILE) -B GOOS=darwin GOARCH=amd64 build
	@${MAKE} -f $(MAKEFILE) -B GOOS=windows GOARCH=amd64 build

deps:
	@${MAKE} -f $(MAKEFILE) -B SCRIPTDIR=$(SCRIPTDIR) BUILDDIR=$(BUILDDIR) -s $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s $(GOBUILDLINKTARGET) $(REPODIR)
	@rm -f $(GOBUILDDIR)/src/github.com/arangodb && ln -s ../../../vendor/github.com/arangodb $(GOBUILDDIR)/src/github.com/arangodb
	@rm -f $(GOBUILDDIR)/src/github.com/pkg && ln -s ../../../vendor/github.com/pkg $(GOBUILDDIR)/src/github.com/pkg
	@rm -f $(GOBUILDDIR)/src/github.com/prometheus && ln -s ../../../vendor/github.com/prometheus $(GOBUILDDIR)/src/github.com/prometheus
	@rm -f $(GOBUILDDIR)/src/github.com/spf13 && ln -s ../../../vendor/github.com/spf13 $(GOBUILDDIR)/src/github.com/spf13
	@rm -f $(GOBUILDDIR)/src/github.com/sirupsen && ln -s ../../../vendor/github.com/sirupsen $(GOBUILDDIR)/src/github.com/sirupsen
	@rm -f $(GOBUILDDIR)/src/gopkg.in && ln -s ../../vendor/gopkg.in $(GOBUILDDIR)/src/gopkg.in

$(BIN): $(GOBUILDDIR) $(SOURCES)
	@mkdir -p $(BINDIR)
	docker run \
		--rm \
		-v $(SRCDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -a -installsuffix netgo -tags netgo -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/bin/$(GOOS)/$(GOARCH)/$(BINNAME) $(REPOPATH)

docker: build
	docker build -t arangodb/arangodb-exporter .

docker-push: docker
ifneq ($(DOCKERNAMESPACE), arangodb)
	docker tag arangodb/arangodb-exporter $(DOCKERNAMESPACE)/arangodb-exporter
endif
	docker push $(DOCKERNAMESPACE)/arangodb-exporter

docker-push-version: docker
	docker tag arangodb/arangodb-exporter arangodb/arangodb-exporter:$(VERSION)
	docker tag arangodb/arangodb-exporter arangodb/arangodb-exporter:$(VERSION_MAJOR_MINOR)
	docker tag arangodb/arangodb-exporter arangodb/arangodb-exporter:$(VERSION_MAJOR)
	docker tag arangodb/arangodb-exporter arangodb/arangodb-exporter:latest
	docker push arangodb/arangodb-exporter:$(VERSION)
	docker push arangodb/arangodb-exporter:$(VERSION_MAJOR_MINOR)
	docker push arangodb/arangodb-exporter:$(VERSION_MAJOR)
	docker push arangodb/arangodb-exporter:latest

$(RELEASE): $(GOBUILDDIR) $(SOURCES) $(GHRELEASE)
	GOPATH=$(GOBUILDDIR) go build -o $(RELEASE) $(REPOPATH)/tools/release

$(GHRELEASE): $(GOBUILDDIR) 
	GOPATH=$(GOBUILDDIR) go build -o $(GHRELEASE) github.com/aktau/github-release

release-patch: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=patch 

release-minor: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=minor

release-major: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=major 

