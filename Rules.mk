TGT_BIN :=
CLEAN :=
DISTCLEAN :=
TEST :=
TEST_SHORT :=

all: help    # all has to be first defined target
.PHONY: all

include mk/util.mk
include mk/golang.mk
include mk/gx.mk

# -------------------- #
#       sub-files      #
# -------------------- #
dir := bin
include $(dir)/Rules.mk

dir := test
include $(dir)/Rules.mk

dir := coverage
include $(dir)/Rules.mk

dir := cmd/ipfs
include $(dir)/Rules.mk

dir := namesys/pb
include $(dir)/Rules.mk

dir := unixfs/pb
include $(dir)/Rules.mk

dir := merkledag/pb
include $(dir)/Rules.mk

dir := exchange/bitswap/message/pb
include $(dir)/Rules.mk

dir := diagnostics/pb
include $(dir)/Rules.mk

dir := pin/internal/pb
include $(dir)/Rules.mk

# -------------------- #
#   universal rules    #
# -------------------- #

%.pb.go: %.proto
	$(PROTOC)

# -------------------- #
#   extra properties   #
# -------------------- #

ifeq ($(TEST_NO_FUSE),1)
	GOTAGS += nofuse
endif
export IPFS_REUSEPORT=false

# -------------------- #
#     core targets     #
# -------------------- #


build: $(TGT_BIN)
.PHONY: build

clean:
	rm -f $(CLEAN)
.PHONY: clean

coverage: $(COVERAGE)

distclean: clean
	rm -f $(DISTCLEAN)
.PHONY: distclean

test: $(TEST)
.PHONY: test

test_short: $(TEST_SHORT)
.PHONY: test_short

deps: gx-deps
.PHONY: deps

nofuse: GOTAGS += nofuse
nofuse: build
.PHONY: nofuse

install: $$(DEPS_GO)
	go install $(go-flags-with-tags) ./cmd/ipfs
.PHONY: install

uninstall:
	go clean -i ./cmd/ipfs
.PHONY: uninstall

help:
	@echo 'DEPENDENCY TARGETS:'
	@echo ''
	@echo '  deps                 - Download dependencies using bundled gx'
	@echo '  test_sharness_deps   - Download and build dependencies for sharness'
	@echo ''
	@echo 'BUILD TARGETS:'
	@echo ''
	@echo '  all          - print this help message'
	@echo '  build        - Build binary at ./cmd/ipfs/ipfs'
	@echo '  nofuse       - Build binary with no fuse support'
	@echo '  install      - Build binary and install into $$GOPATH/bin'
#	@echo '  dist_install - TODO: c.f. ./cmd/ipfs/dist/README.md'
	@echo ''
	@echo 'CLEANING TARGETS:'
	@echo ''
	@echo '  clean        - Remove files generated by build'
	@echo '  distclean    - Remove files that are no part of a repository'
	@echo '  uninstall    - Remove binary from $$GOPATH/bin'
	@echo ''
	@echo 'TESTING TARGETS:'
	@echo ''
	@echo '  test                    - Run expensive tests'
	@echo '  test_short              - Run short tests and short sharness tests'
	@echo '  test_go_short'
	@echo '  test_go_expensive'
	@echo '  test_go_race'
	@echo '  test_sharness_short'
	@echo '  test_sharness_expensive'
	@echo '  test_sharness_race'
	@echo '  coverage     - Collects coverage info from unit tests and sharness'
	@echo
.PHONY: help
