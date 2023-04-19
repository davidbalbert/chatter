#!/bin/sh

# set -eo pipefail

version=$(cat VERSION)

if [ -z "$version" ]; then
    echo "unknown"
    exit 0
fi

git_version=$(git describe --exact-match --tags 2>/dev/null || true)
sha=$(git rev-parse --short HEAD)

dirty=""
if [ -n "$(git status --porcelain)" ]; then
    dirty="; dirty"
fi

if [ "$version" = "$git_version" ]; then
    printf "$version"
else
    printf "$version-dev"
fi

if [ "$1" = "--skip-hash" ]; then
    printf "\n"
else
    printf " ($sha$dirty)\n"
fi
