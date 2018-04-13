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

ifndef DOCKERNAMESPACE
	echo "Set DOCKERNAMESPACE"
	exit 1
endif
ifndef DOCKERTAG 
	DOCKERTAG := dev
endif
DOCKERIMAGE := $(DOCKERNAMESPACE)/arangodb-exporter:$(DOCKERTAG)

PULSAR := $(GOBUILDDIR)/bin/pulsar$(shell go env GOEXE)
RELEASE := $(GOBUILDDIR)/bin/release$(shell go env GOEXE)
GHRELEASE := $(GOBUILDDIR)/bin/github-release$(shell go env GOEXE)
GOX := $(GOBUILDDIR)/bin/gox$(shell go env GOEXE)
MANIFESTTOOL := $(GOBUILDDIR)/bin/manifest-tool$(shell go env GOEXE)

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
	rm -Rf $(BINDIR) $(GOBUILDDIR) $(ROOTDIR)/arangodb-exporter

build: $(GOBUILDDIR) $(GHRELEASE) $(GOX) $(MANIFESTTOOL)
	CGO_ENABLED=0 GOPATH=$(GOBUILDDIR) $(GOX) \
		-os="darwin linux windows" \
		-arch="$(ARCHS)" \
		-osarch="!darwin/arm !darwin/arm64" \
		-ldflags="-X main.projectVersion=${VERSION} -X main.projectBuild=${COMMIT}" \
		-output="bin/{{.OS}}/{{.Arch}}/arangodb-exporter" \
		-tags="netgo" \
		github.com/arangodb-helper/arangodb-exporter
	@ln -sf $(BINDIR)/$(shell go env GOOS)/$(shell go env GOARCH)/arangodb-exporter$(shell go env GOEXE)

$(GOBUILDDIR):
	# Build pulsar from vendor
	@mkdir -p $(GOBUILDDIR)
	@ln -sf $(VENDORDIR) $(GOBUILDDIR)/src
	@GOPATH=$(GOBUILDDIR) go install github.com/pulcy/pulsar
	@rm -Rf $(GOBUILDDIR)/src
	# Prepare .gobuild directory
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -sf ../../../.. $(REPODIR)
	GOPATH=$(GOBUILDDIR) $(PULSAR) go flatten -V $(VENDORDIR)

.PHONY: update-vendor
update-vendor:
	@mkdir -p $(GOBUILDDIR)
	@GOPATH=$(GOBUILDDIR) go get github.com/pulcy/pulsar
	@rm -Rf $(VENDORDIR)
	@mkdir -p $(VENDORDIR)
	@$(PULSAR) go vendor -V $(VENDORDIR) \
		github.com/aktau/github-release \
		github.com/arangodb/go-driver \
		github.com/coreos/go-semver/semver \
		github.com/dgrijalva/jwt-go \
		github.com/estesp/manifest-tool \
		github.com/mitchellh/gox \
		github.com/pkg/errors \
		github.com/prometheus/client_golang/prometheus \
		github.com/pulcy/pulsar \
		github.com/spf13/cobra
	@$(PULSAR) go flatten -V $(VENDORDIR) $(VENDORDIR)
	@${MAKE} -B -s clean

docker: build
	for arch in $(ARCHS); do \
		docker build --build-arg=GOARCH=$$arch -t $(DOCKERIMAGE)-$$arch . ;\
		docker push $(DOCKERIMAGE)-$$arch ;\
	done
	$(MANIFESTTOOL) $(MANIFESTAUTH) push from-args \
    	--platforms $(PLATFORMS) \
    	--template $(DOCKERIMAGE)-ARCH \
    	--target $(DOCKERIMAGE)

$(RELEASE): $(GOBUILDDIR) $(SOURCES) $(GHRELEASE)
	GOPATH=$(GOBUILDDIR) go build -o $(RELEASE) $(REPOPATH)/tools/release

$(GHRELEASE): $(GOBUILDDIR) 
	GOPATH=$(GOBUILDDIR) go build -o $(GHRELEASE) github.com/aktau/github-release

$(GOX): 
	GOPATH=$(GOBUILDDIR) go build -o $(GOX) github.com/mitchellh/gox

$(MANIFESTTOOL): 
	GOPATH=$(GOBUILDDIR) go build -o $(MANIFESTTOOL) github.com/estesp/manifest-tool


release-patch: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=patch 

release-minor: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=minor

release-major: $(RELEASE)
	GOPATH=$(GOBUILDDIR) $(RELEASE) -type=major 

