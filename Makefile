CMD := chatterd chatterc
BUILD_DIR := build

# e.g. "build/chatterd build/chatterc"
TARGETS := $(patsubst %,$(BUILD_DIR)/%,$(CMD))

VERSION := $(shell script/version.sh)
LDFLAGS := -ldflags '-X "main.version=$(VERSION)"'

.PHONY: all clean
all: $(TARGETS)

clean: 
	rm -rf $(TARGETS)

$(BUILD_DIR)/%: %/*.go %/**/*.go
	go build $(LDFLAGS) -o $@ ./$*
