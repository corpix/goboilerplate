.DEFAULT_GOAL := all

## parameters

name              ?= goboilerplate
namespace         ?= git.backbone/corpix
version           ?= development
os                ?=
binary            ?= ./main

PARALLEL_JOBS ?= 8
NIX_OPTS      ?=

export GOFLAGS ?=

-include .env

## bindings

root                := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
nix_dir             := nix
pkg_prefix          := $(namespace)/$(name)
tmux                := tmux -2 -f $(PWD)/.tmux.conf -S $(PWD)/.tmux
tmux_session        := $(name)
nix                 := nix $(NIX_OPTS)
shell_volume_nix    := nix

### reusable and long opts for commands inside rules

shell_opts ?=
docker_shell_opts = -v $(shell_volume_nix):/nix:rw  \
	-v $(root):/chroot                          \
	-e COLUMNS=$(COLUMNS)                       \
	-e LINES=$(LINES)                           \
	-e TERM=$(TERM)                             \
	-e NIX_BUILD_CORES=$(NIX_BUILD_CORES)       \
	-e HOME=/chroot                             \
	-w /chroot                                  \
	--hostname $(namespace).localhost           \
	$(foreach v,$(ports), -p $(v):$(v) ) $(shell_opts)

## helpers

, = ,

## macro

define fail
{ echo "error: "$(1) 1>&2; exit 1; }
endef

## targets

.PHONY: all
all: build # test, check and build all cmds

.PHONY: help
help: # print defined targets and their comments
	@grep -Po '^[a-zA-Z%_/\-\s]+:+(\s.*$$|$$)' Makefile \
		| sort                                      \
		| sed 's|:.*#|#|;s|#\s*|#|'                 \
		| column -t -s '#' -o ' | '

### releases

### development

.PHONY: build
build $(binary): # build application `binary`
	GOOS=$(os)                                                  \
	go build -o $(binary)                                       \
		--ldflags "-X $(pkg_prefix)/cli.Version=$(version)" \
		./main.go

.PHONY: run
run: # run application
	go run ./main.go

#### testing

.PHONY: test
test: # run unit tests
	go test -v ./...

.PHONY: lint
lint: # run linter
	golangci-lint --color=always --timeout=120s run ./...

.PHONY: profile
profile: build # collect profile for `binary`
	$(binary) --profile

.PHONY: pprof
pprof: $(binary) # run pprof web server to visualize collected `profile`
	@[ -z "$(profile)" ] && $(call fail,"profile=<value> parameter is required$(,) available values: cpu$(,) heap") || true
	go tool pprof -http=":8081" $(binary) $(profile).prof

#### environment management

.PHONY: dev/clean
dev/clean: # clean development environment artifacts
	docker volume rm nix

.PHONY: dev/shell
dev/shell: # run development environment shell
	@docker run --rm -it                          \
		--log-driver=none                     \
		$(docker_shell_opts) nixos/nix:latest \
		nix-shell --command "exec make dev/start-session"

.PHONY: dev/shell/raw
dev/shell/raw: # run development environment shell
	@docker run --rm -it                          \
		--log-driver=none                     \
		$(docker_shell_opts) nixos/nix:latest \
		nix-shell

.PHONY: dev/session
dev/start-session: # start development environment terminals with database, blockchain, etc... one window per app
	@$(tmux) has-session    -t $(tmux_session) && $(call fail,tmux session $(tmux_session) already exists$(,) use: '$(tmux) attach-session -t $(tmux_session)' to attach) || true
	@$(tmux) new-session    -s $(tmux_session) -n console -d
	@sleep 1 # sometimes input is messed up (bash+stdin early handling?)
	@$(tmux) select-window  -t $(tmux_session):0

	@if [ -f $(root)/.personal.tmux.conf ]; then             \
		$(tmux) source-file $(root)/.personal.tmux.conf; \
	fi

	@$(tmux) attach-session -t $(tmux_session)

.PHONY: dev/attach-session
dev/attach-session: # attach to development session if running
	@$(tmux) attach-session -t $(tmux_session)

.PHONY: dev/stop-session
dev/stop-session: # stop development environment terminals
	@$(tmux) kill-session -t $(tmux_session)

.PHONY: clean
clean: # clean stored state
	rm -rf result*
