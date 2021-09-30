let GitHubActions = (../imports.dhall).GitHubActions

let Setup = ../setup.dhall

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
