# Sourcegraph resource estimator

Sourcegraph resource estimator helps predict and plan the required resource for your deployment.

This tool ensures you provision appropriate resources to scale your instance.

## Development Prerequisites

- [Install Go](https://golang.org/doc/install), then:

### Optional prerequisites

The project includes a [justfile](https://github.com/sourcegraph/infrastructure/blob/main/justfile) that includes simple
commands for automating common tasks such as:

- Building the WASM binary: `just build`
- Automatic formatting of various file types: `just format`
- Linting of various filetypes: `just lint`
- A single command to perform all of the above steps: `just`

You can install `just` and the various commands it calls by either:

1. Separately installing each of the tools listed in [.tool-versions](./.tool-versions) (via homebrew or something similar)
2. Using the [asdf version manager](https://github.com/asdf-vm/asdf) and running [scripts/asdf-add-plugins.sh](./scripts/asdf-add-plugins.sh) followed by `asdf install`

## Development

```sh
./scripts/watch-wasm.sh # or "just watch | just dev"
```

This will start serving on http://localhost:8080 the WASM bundle and recompiling code each time you reload the page (any errors compiling your changes will show up in this terminal).

### Developing on the docsite

Visit http://docs.sourcegraph.com/admin/install/resource_estimator?dev=true and the page will use your local `wasmserve` instance, so
that changes you make to the code are automatically reflected when you reload the page.

This is the best option as you can see the page with full CSS styling.

### Developing without the docsite

You can view the page without any CSS styling at http://localhost:8080

### Golden test

Run `go test -update` in the internal/scaling directory to update the tests

## Releasing

1. Get your changes merged into `master` first and `git checkout master`.
2. Run `./build.sh` which will produce a `main_$COMMIT.wasm` file.
3. Upload that file to the [Google Cloud Storage bucket](https://console.cloud.google.com/storage/browser/sourcegraph-resource-estimator?authuser=1&project=sourcegraph-dev).
4. Update the `version=<oldcommit>` attribute [in this file](https://github.com/sourcegraph/sourcegraph/edit/master/doc/admin/install/resource_estimator.md) to match the file you uploaded.
5. Merge that PR and your change has been released!
