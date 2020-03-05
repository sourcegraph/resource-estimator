# resource-estimator

Sourcegraph resource estimator

## Development Prerequisites

- [Install Go](https://golang.org/doc/install), then:
- Install `wasmserve`:

```sh
go get -u github.com/hajimehoshi/wasmserve
```

## Development

```sh
cd resource-estimator/
export GOROOT=$(go env GOROOT)
wasmserve -allow-origin='*'
```

This will start serving on http://localhost:8080 the WASM bundle and recompiling code each time you reload the page (any errors compiling your changes will show up in this terminal).

### Developing on the docsite

Visit http://docs.sourcegraph.com/admin/install/resource_estimator?dev=true and the page will use your local `wasmserve` instance, so
that changes you make to the code are automatically reflected when you reload the page.

This is the best option as you can see the page with full CSS styling.

### Developing without the docsite

You can view the page without any CSS styling at http://localhost:8080

## Releasing

1. Get your changes merged into `master` first and `git checkout master`.
2. Run `./build.sh` which will produce a `main_$COMMIT.wasm` file.
3. Upload that file to the [Google Cloud Storage bucket](https://console.cloud.google.com/storage/browser/sourcegraph-resource-estimator?authuser=1&project=sourcegraph-dev).
4. Update the `version=<oldcommit>` attribute [in this file](https://github.com/sourcegraph/sourcegraph/edit/master/doc/admin/install/resource_estimator.md) to match the file you uploaded.
5. Merge that PR and your change has been released!
