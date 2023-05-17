REPO_ROOT:=${CURDIR}/..
# setup go for managing hugo
PATH:=$(shell cd $(REPO_ROOT) && . hack/build/setup-go.sh && echo "$${PATH}")
# go1.9+ can autodetect GOROOT, but if some other tool sets it ...
GOROOT:=
# enable modules
GO111MODULE=on
# disable CGO by default for static binaries
CGO_ENABLED=0
export PATH GOROOT GO111MODULE CGO_ENABLED
# work around broken PATH export
SPACE:=$(subst ,, )
SHELL:=env PATH=$(subst $(SPACE),\$(SPACE),$(PATH)) $(SHELL)

# from https://github.com/kubernetes/website/blob/master/Makefile
DOCKER       = docker
HUGO_VERSION = 0.60.0
DOCKER_IMAGE = jojomi/hugo:$(HUGO_VERSION)
DOCKER_RUN   = $(DOCKER) run --rm --interactive --tty --volume $(realpath $(CURDIR)/..):/src -p 1313:1313 --workdir /src/site --entrypoint=hugo --platform linux/amd64 $(DOCKER_IMAGE)

HUGO_BIN:=$(REPO_ROOT)/bin/hugo

$(HUGO_BIN):
	go build -o $(HUGO_BIN) github.com/gohugoio/hugo

hugo: $(HUGO_BIN)

serve: hugo
	$(HUGO_BIN) server --bind="0.0.0.0" \
	--ignoreCache \
	--buildFuture \
	--disableFastRender

build: hugo
	$(HUGO_BIN)

.PHONY: build serve hugo
