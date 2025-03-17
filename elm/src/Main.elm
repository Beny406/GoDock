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
    , instances : Maybe (List WmInstance)
    , execPath : String
    , wmClass : String
    }


type alias WmCtrlApp =
    { wmClass : String
    , instances : List WmInstance
    }


type alias WmInstance =
    { name : String
    , windowId : String
    }


type alias AppInfo =
    { iconPath : String
    , name : String
    , instances : Maybe (List WmInstance)
    , execPath : String
    , wmClass : String
    , justClicked : Maybe String
    }


type alias Model =
    { desktopApps : List AppInfo
    , hoveredClass : Maybe String
    }


init : Flags -> ( Model, Cmd msg )
init { apps } =
    ( { desktopApps =
            apps
                |> List.map
                    (\flagApp ->
                        { iconPath = flagApp.iconPath
                        , name = flagApp.name
                        , instances = flagApp.instances
                        , execPath = flagApp.execPath
                        , wmClass = flagApp.wmClass
                        , justClicked = Nothing
                        }
                    )
      , hoveredClass = Nothing
      }
    , Cmd.none
    )



-- UPDATE


type Msg
    = IconClicked AppInfo (Maybe String)
    | BouncingRunOut AppInfo
    | RunningAppsReceived (List WmCtrlApp)
    | ClassMouseLeave String
    | ClassMouseEnter String
    | AppMouseLeave


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        IconClicked app maybeId ->
            { model | desktopApps = List.updateIf (\app_ -> app.name == app_.name) (always app) model.desktopApps }
                |> Return.singleton
                |> Return.command (iconClicked ( maybeId, app.execPath ))
                |> Return.pushMsgWithTimeout 500 (BouncingRunOut { app | justClicked = Nothing })

        BouncingRunOut app ->
            { model | desktopApps = List.updateIf (\app_ -> app.name == app_.name) (always app) model.desktopApps }
                |> Return.singleton

        RunningAppsReceived newRunningCtrlApps ->
            { model
                | desktopApps =
                    model.desktopApps
                        |> List.map
                            (\app ->
                                let
                                    newRunningIds : Maybe (List WmInstance)
                                    newRunningIds =
                                        newRunningCtrlApps
                                            |> List.find (\newApp -> newApp.wmClass == app.wmClass)
                                            |> Maybe.map .instances
                                in
                                { app | instances = newRunningIds }
                            )
            }
                |> Return.singleton

        ClassMouseLeave string ->
            if model.hoveredClass == Just string then
                { model | hoveredClass = Nothing }
                    |> Return.singleton

            else
                model
                    |> Return.singleton

        ClassMouseEnter string ->
            { model | hoveredClass = Just string }
                |> Return.singleton

        AppMouseLeave ->
            model
                |> Return.singleton
                |> Return.command (mouseAppLeft ())



-- VIEW


view : Model -> H.Html Msg
view model =
    H.div [ HA.id "app" ]
        [ flexColumn
            [ HA.css
                [ Css.justifyContent Css.spaceBetween
                , Css.alignItems Css.start
                , Css.padding2 (Css.px 8) (Css.px 4)
                , Css.minWidth Css.zero
                , Css.maxWidth Css.minContent
                ]
            , HE.onMouseLeave AppMouseLeave
            ]
            (model.desktopApps
                |> List.map
                    (\app ->
                        H.div
                            [ HA.id app.wmClass
                            , HE.onMouseEnter (ClassMouseEnter app.wmClass)
                            , HE.onMouseLeave (ClassMouseLeave app.wmClass)
                            ]
                            [ case app.instances of
                                Nothing ->
                                    H.div
                                        [ HE.onClick (IconClicked { app | justClicked = Just "" } Nothing)
                                        , HA.classList
                                            [ ( "icon-container", True )
                                            , ( "bounce", Maybe.isJust app.justClicked )
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

                                Just instances ->
                                    (if model.hoveredClass == Just app.wmClass then
                                        instances

                                     else
                                        instances |> List.take 1
                                    )
                                        |> (\instances_ ->
                                                flexRow [ HA.css [ Css.justifyContent Css.center, Css.marginRight (Css.px 16) ] ]
                                                    (instances_
                                                        |> List.map
                                                            (\instance ->
                                                                H.div
                                                                    [ HE.onClick (IconClicked { app | justClicked = Just instance.windowId } (Just instance.windowId))
                                                                    , HA.classList
                                                                        [ ( "icon-container", True )
                                                                        , ( "running", True )
                                                                        , ( "bounce", app.justClicked == Just instance.windowId )
                                                                        ]
                                                                    ]
                                                                    [ H.img
                                                                        [ HA.class "icon"
                                                                        , HA.width 64
                                                                        , HA.height 64
                                                                        , HA.src app.iconPath
                                                                        , HA.title instance.name
                                                                        ]
                                                                        []
                                                                    ]
                                                            )
                                                    )
                                           )
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


port mouseAppLeft : () -> Cmd msg


port iconClicked : ( Maybe String, String ) -> Cmd msg


port runningAppsReceived : (List WmCtrlApp -> msg) -> Sub msg
