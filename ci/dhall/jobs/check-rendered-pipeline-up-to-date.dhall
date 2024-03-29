let Setup = ../setup.dhall

let GitHubActions = (../imports.dhall).GitHubActions

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "render-CI-pipeline"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some
              "Check that the CI pipeline definition is up-to-date with the dhall configuration"
          , run = Some "ci/check-rendered-pipeline-up-to-date.sh"
          }
        ]
      }
