port module Main exposing (..)

import Browser
import Css
import Html.Styled as H
import Html.Styled.Attributes as HA
import Html.Styled.Events as HE
import List.Extra as List
import Maybe.Extra as Maybe
import Return
import Shared.Return.Extra as Return



-- MAIN


main : Program Flags Model Msg
main =
    Browser.element
        { init = init
        , update = update
        , view = view >> H.toUnstyled
        , subscriptions = subscriptions
        }



-- MODEL


type alias Flags =
    { apps : List AppInfoFromFlags
    }


type alias AppInfoFromFlags =
    { iconPath : String
    , name : String
    , runningId : Maybe String
    , execPath : String
    , wmClass : String
    }


type alias AppInfo =
    { iconPath : String
    , name : String
    , runningId : Maybe String
    , execPath : String
    , wmClass : String
    , justClicked : Bool
    }


type alias Model =
    { apps : List AppInfo
    }


init : Flags -> ( Model, Cmd msg )
init { apps } =
    ( { apps =
            apps
                |> List.map
                    (\flagApp ->
                        { iconPath = flagApp.iconPath
                        , name = flagApp.name
                        , runningId = flagApp.runningId
                        , execPath = flagApp.execPath
                        , wmClass = flagApp.wmClass
                        , justClicked = False
                        }
                    )
      }
    , Cmd.none
    )



-- UPDATE


type Msg
    = IconClicked AppInfo
    | BouncingRunOut AppInfo
    | RunningAppsReceived (List ( String, String ))


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        IconClicked app ->
            { model | apps = List.updateIf (\app_ -> app.name == app_.name) (always app) model.apps }
                |> Return.singleton
                |> Return.command (iconClicked ( app.runningId, app.execPath ))
                |> Return.pushMsgWithTimeout 500 (BouncingRunOut { app | justClicked = False })

        BouncingRunOut app ->
            { model | apps = List.updateIf (\app_ -> app.name == app_.name) (always app) model.apps }
                |> Return.singleton

        RunningAppsReceived runningApps ->
            { model
                | apps =
                    model.apps
                        |> List.map
                            (\app ->
                                let
                                    newRunningId : Maybe String
                                    newRunningId =
                                        runningApps
                                            |> List.find (\( wmClass, _ ) -> wmClass == app.wmClass)
                                            |> Maybe.map Tuple.second
                                in
                                { app | runningId = newRunningId }
                            )
            }
                |> Return.singleton



-- VIEW


view : Model -> H.Html Msg
view model =
    H.div [ HA.id "app" ]
        [ flexColumn [ HA.css [ Css.justifyContent Css.spaceBetween, Css.padding2 (Css.px 8) (Css.px 4) ] ]
            (model.apps
                |> List.map
                    (\app ->
                        H.div
                            [ HE.onClick (IconClicked { app | justClicked = True })
                            , HA.classList
                                [ ( "icon-container", True )
                                , ( "running", Maybe.isJust app.runningId )
                                , ( "bounce", app.justClicked )
                                ]
                            ]
                            [ H.img
                                [ HA.class "icon"
                                , HA.width 64
                                , HA.height 64
                                , HA.src app.iconPath
                                , HA.title app.name
                                ]
                                []
                            ]
                    )
            )
        ]


flexColumn : List (H.Attribute msg) -> List (H.Html msg) -> H.Html msg
flexColumn =
    H.styled H.div
        [ Css.displayFlex
        , Css.flexDirection Css.column
        ]


flexRow : List (H.Attribute msg) -> List (H.Html msg) -> H.Html msg
flexRow =
    H.styled H.div
        [ Css.displayFlex
        , Css.flexDirection Css.row
        , Css.minWidth (Css.px 0)
        ]



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    runningAppsReceived RunningAppsReceived



-- PORTS


port iconClicked : ( Maybe String, String ) -> Cmd msg


port runningAppsReceived : (List ( String, String ) -> msg) -> Sub msg
