#!/bin/bash -xe

cd "$(dirname "$(realpath "$0")")/.." || exit 1

ln -sf ../../script/githooks.sh .git/hooks/pre-commit
ln -sf ../../script/githooks.sh .git/hooks/pre-push
