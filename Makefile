TESTS := $(wildcard tests/*.sh)
COMMIT_ID := $(shell git rev-list -1 HEAD) # get the most recent commit
VERSION := "0.1.0"

build:
	@go build -ldflags "-X main.version=${VERSION}-${COMMIT_ID}" .
	@ln -sf simple-linker ld

test: build
	@CC="riscv64-linux-gnu-gcc"
	@$(MAKE) $(TESTS)

$(TESTS):
	@echo -----Testing $@ Start-----
	@./$@
	@echo -----'Testing' $@ Done------

clean:
	go clean
	rm -rf out/

.PHONY: build clean test $(TESTS)