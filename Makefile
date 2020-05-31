GOPKG ?=	moul.io/bot
DOCKER_IMAGE ?=	moul/bot
GOBINS ?=	./cmd/moul-bot

PRE_INSTALL_STEPS += generate
PRE_UNITTEST_STEPS += generate
PRE_TEST_STEPS += generate
PRE_BUILD_STEPS += generate
PRE_LINT_STEPsS += generate
PRE_TIDY_STEPS += generate
PRE_BUMPDEPS_STEPS += generate

include rules.mk

.PHONY: run
run: install
	moul-bot --dev-mode --enable-server --enable-discord run

.PHONY: run-discord
run-discord: install
	moul-bot --dev-mode --enable-discord run

.PHONY: run-server
run-server: install
	moul-bot --dev-mode --enable-server run

PROTOS_SRC := $(wildcard ./api/*.proto)
GEN_DEPS := $(PROTOS_SRC) Makefile
.PHONY: generate
generate: gen.sum
gen.sum: $(GEN_DEPS)
	shasum $(GEN_DEPS) | sort > gen.sum.tmp
	@diff -q gen.sum gen.sum.tmp || ( \
	  set -xe; \
	  GO111MODULE=on go mod vendor; \
	  docker run \
	    --user=`id -u` \
	    --volume="$(PWD):/go/src/moul.io/bot" \
	    --workdir="/go/src/moul.io/bot" \
	    --entrypoint="sh" \
	    --rm \
	    moul/moul-bot-protoc:1 \
	    -xec 'make generate_local'; \
	    make tidy \
	)
	@rm -f gen.sum.tmp

PROTOC_OPTS = -I ./api:/protobuf
.PHONY: generate_local
generate_local:
	@set -e; for proto in $(PROTOS_SRC); do ( set -xe; \
	  protoc $(PROTOC_OPTS) \
	    --grpc-gateway_out=logtostderr=true:"$(GOPATH)/src" \
	    --go_out="$(GOPATH)/src" \
	    --go-grpc_out="$(GOPATH)/src" \
	    "$$proto" \
	); done
	goimports -w ./pkg ./cmd ./internal
	shasum $(GEN_SRC) | sort > gen.sum.tmp
	mv gen.sum.tmp gen.sum

.PHONY: clean
clean:
	rm -f gen.sum $(wildcard */*/*.pb.go */*/*.pb.gw.go)
	@# packr
