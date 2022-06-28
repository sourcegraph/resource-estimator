let Setup = ../setup.dhall

let GitHubActions = (../imports.dhall).GitHubActions

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "dhall-lint"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some "Check that dhall files are linted properly"
          , run = Some "just lint-dhall"
          , env = Some (toMap { CHECK = "true" })
          }
        ]
      }
