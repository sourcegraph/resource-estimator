let shellcheck = ./jobs/shellcheck.dhall

let shfmt = ./jobs/shfmt.dhall

let checkPipeline = ./jobs/check-rendered-pipeline-up-to-date.dhall

let dhallFormat = ./jobs/dhall-format.dhall

let dhallLint = ./jobs/dhall-lint.dhall

let dhallCheck = ./jobs/dhall-check.dhall

let prettier = ./jobs/prettier.dhall

let golangciLint = ./jobs/golangci-lint.dhall

let wasmBuild = ./jobs/wasm-build.dhall

let GitHubActions = (./imports.dhall).GitHubActions

in  GitHubActions.Workflow::{
    , name = "CI"
    , on = GitHubActions.On::{ push = Some GitHubActions.Push::{=} }
    , jobs = toMap
        { shellcheck
        , shfmt
        , dhallCheck
        , dhallFormat
        , dhallLint
        , checkPipeline
        , prettier
        , golangciLint
        , wasmBuild
        }
    }
