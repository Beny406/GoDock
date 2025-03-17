module Shared.Return.Extra exposing (pushMsgWithTimeout)

import Process
import Return exposing (Return)
import Task


pushMsgWithTimeout : Float -> msg -> Return msg model -> Return msg model
pushMsgWithTimeout delay msg ret =
    Return.command
        (delay
            |> Process.sleep
            |> Task.andThen (\_ -> Task.succeed msg)
            |> Task.perform identity
        )
        ret
