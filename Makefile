SHELL := /bin/bash
.PHONY: compile build test clean release devshell version

usage:
	@echo "USAGE:"
	@echo "   make command [options]"
	@echo
	@echo "COMMANDS:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed 's/^/   /' | sed -e 's/\\$$/AA/' | sed -e 's/#//g' | column -t -s ":" | sort -k1



run: ## run locally
	go run cmd/rsssplit/main.go --config=config.yaml

build: bump_version
	./scripts/docker_build.sh
bump_version:
	./scripts/bump_version.sh
push:
	./scripts/docker_push.sh
