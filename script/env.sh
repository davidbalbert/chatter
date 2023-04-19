#!/bin/sh

set -eo pipefail

# The full version has a space, and I can't get that to work with GOFLAGS.
VERSION=$(./script/version.sh --skip-hash)

cat <<END > .env
GOFLAGS=-ldflags=-X=main.version=$VERSION
END
