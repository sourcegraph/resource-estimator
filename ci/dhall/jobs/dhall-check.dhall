let Setup = ../setup.dhall

let GitHubActions = (../imports.dhall).GitHubActions

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "dhall-check"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some "Check that all dhall files typecheck"
          , run = Some "just check-dhall"
          }
        ]
      }
