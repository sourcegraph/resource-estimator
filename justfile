all: format lint build

build: build-wasm

fmt: format

format: prettier format-shfmt

lint: shellcheck

build-wasm:
    ./build.sh

dev: watch

watch: watch-wasm
watch-wasm:
    ./scripts/watch-wasm.sh

prettier:
    yarn run prettier

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
