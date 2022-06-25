let Setup = ../setup.dhall

let GitHubActions = (../imports.dhall).GitHubActions

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "shellcheck"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some "Lint shell scripts"
          , run = Some "just shellcheck"
          }
        ]
      }
