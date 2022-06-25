let Setup = ../setup.dhall

let GitHubActions = (../imports.dhall).GitHubActions

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "Prettier formatting"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some "Check Prettier formatting"
          , run = Some "ci/check-prettier.sh"
          }
        ]
      }
