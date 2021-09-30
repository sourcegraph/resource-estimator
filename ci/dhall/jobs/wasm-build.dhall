let GitHubActions = (../imports.dhall).GitHubActions

let Setup = ../setup.dhall

in  Setup.MakeJob
      Setup.JobArgs::{
      , name = "Build WASM"
      , additionalSteps =
        [ GitHubActions.Step::{
          , name = Some "Build WASM binary"
          , run = Some "just build-wasm"
          , env = Some (toMap { CI = "true" })
          }
        ]
      }
