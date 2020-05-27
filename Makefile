GOPKG ?=	moul.io/moul-bot
DOCKER_IMAGE ?=	moul/moul-bot
GOBINS ?=	./cmd/moul-bot

include rules.mk

.PHONY: run-discord
run-discord: install
	moul-bot discord-bot
