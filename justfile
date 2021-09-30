all: check format lint build

build: render-ci-pipeline build-wasm

render-ci-pipeline:
    ./scripts/render-ci-pipeline.sh

fmt: format

format: format-dhall prettier format-shfmt

lint: lint-dhall shellcheck

check: check-dhall

build-wasm:
    ./build.sh

dev: watch

watch: watch-wasm
watch-wasm:
    ./scripts/watch-wasm.sh

prettier:
    yarn run prettier

format-dhall:
    ./scripts/dhall-format.sh

check-dhall:
    ./scripts/dhall-check.sh

lint-dhall:
    ./scripts/dhall-lint.sh

shellcheck:
    ./scripts/shellcheck.sh

format-shfmt:
    shfmt -w .

install:
    just install-asdf
    just install-yarn

install-yarn:
    yarn

install-asdf:
    asdf install
