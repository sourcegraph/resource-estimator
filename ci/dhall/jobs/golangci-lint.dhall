let Setup = ../setup.dhall

let List/map =
      https://raw.githubusercontent.com/dhall-lang/dhall-lang/v21.0.0/Prelude/List/map
        sha256:dd845ffb4568d40327f2a817eb42d1c6138b929ca758d50bc33112ef3c885680

let GitHubActions = (../imports.dhall).GitHubActions

let MakeSteps
    -- I should refactor this at some point to create a separate job or workflow
    : ∀(dirs : List Text) → List GitHubActions.Step.Type
    = λ(dirs : List Text) →
        List/map
          Text
          GitHubActions.Step.Type
          ( λ(d : Text) →
              GitHubActions.Step::{
              , name = Some "golangci-lint (${d})"
              , uses = Some "golangci/golangci-lint-action@v3"
              , `with` = Some
                  (toMap { version = "v1.46.2", working-directory = d })
              }
          )
          dirs

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "golangci-lint"
      , additionalSteps = MakeSteps [ "." ]
      }
