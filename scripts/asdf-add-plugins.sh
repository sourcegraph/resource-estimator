#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"/..
set -euxo pipefail

OTHER_PACKAGES=(
  "dhall"
  "shellcheck"
  "shfmt"
  "fd"
  "yarn"
  "deno"
  "golang"
  "just"
  "nodejs"
)

for package in "${OTHER_PACKAGES[@]}"; do
  asdf plugin-add "${package}"
done
