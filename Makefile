# SyntropyAgent-GO build script

APPNAME:=syntropy_agent
# Get git discribe. Github actions will pass this variable.
# If it is missing - then this is a local build and get it from git.
# AGENT_VERSION is set by Docker build
FULL_VERSION := $(AGENT_VERSION)
ifeq ($(FULL_VERSION), ${EMPTY:Q})
FULL_VERSION := $(shell git describe --tags --dirty --candidates=1)
endif
# Split git describe into version and subversion
# 1.0.4-14-g2414721-dirty ==> version = 1.0.4, subversion = 14.g2414721.dirty
# NOTE: do not include `v` in versioning
VERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f1)
SUBVERSION = $(shell echo $(FULL_VERSION) | cut -d "-" -f2-6 | sed -r 's/-/+/' | sed -r 's/-/./g')
ifeq ($(FULL_VERSION), $(VERSION))
SUBVERSION:=
endif

# Sanity fallback (should not happen in normal environment)
ifeq ($(VERSION), ${EMPTY:Q})
VERSION:=0.0.0
endif

destdir = target/$(shell uname -m)

all: deps syntropy_agent

deps:
	@echo Fetching dependencies:
	go get -d ./...

destdir:
	mkdir -p $(destdir)

syntropy_agent:
	@echo Building $(APPNAME)  $(VERSION) - $(SUBVERSION)
# build the agent
	CGO_ENABLED=0 go build -o $(APPNAME) -ldflags \
		"-X github.com/SyntropyNet/syntropy-agent/internal/config.version=$(VERSION) \
		-X github.com/SyntropyNet/syntropy-agent/internal/config.subversion=$(SUBVERSION) -s -w" \
		./cmd/main.go

$(destdir)/wireguard-go: destdir
	git clone https://git.zx2c4.com/wireguard-go && \
	cd wireguard-go && \
	git checkout $(git describe --tags $(git rev-list --tags --max-count=1)) && \
	CGO_ENABLED=0 make && \
	cp wireguard-go ../$(destdir)
	rm -rf wireguard-go

wireguard: $(destdir)/wireguard-go

docker: destdir deps syntropy_agent wireguard
	cp $(APPNAME) $(destdir)
	docker build . -t syntropynet/agent


test:
	go test ./...

clean:
	go clean
	rm -f $(APPNAME)

distclean: clean
	rm -rf target
	rm -rf wireguard-go
