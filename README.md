# GoDockElm

A lightweight application dock for Linux (X11), built with [Wails](https://wails.io) (Go backend) and [Elm](https://elm-lang.org) (frontend).

The dock sits on the left edge of the screen and slides into view when the mouse reaches the left side near the vertical center. Clicking an icon launches the app, focuses a running instance, or minimizes it on a second click.

```
Mouse → left edge
           |
           v
    +------+
    | icon |  <- running (highlighted)
    | icon |  <- not running
    | icon |  <- running, hovered shows all instances
    +------+
```

## Requirements

- Linux with X11
- [Go toolchain](https://go.dev/doc/install)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

## Configuration

Create `~/GoDock/apps.json` listing the apps you want in the dock:

```json
[
  {
    "name": "firefox",
    "iconPath": "/usr/share/icons/hicolor/64x64/apps/firefox.png",
    "execPath": "firefox",
    "wmClass": "firefox"
  },
  {
    "name": "Thunar",
    "iconPath": "/usr/share/icons/hicolor/48x48/apps/Thunar.png",
    "execPath": "thunar",
    "wmClass": "thunar"
  }
]
```

| Field      | Description                                                   |
|------------|---------------------------------------------------------------|
| `name`     | Display name (also used for window matching)                  |
| `iconPath` | Absolute path to the app icon                                 |
| `execPath` | Command to launch the app                                     |
| `wmClass`  | WM_CLASS instance name — run `wmctrl -lx` to find the value  |

## Running

```bash
wails dev
```

## Building

```bash
wails build
```

Pre-built binaries for each platform are available on the [Releases](../../releases) page.

## Autostart

To launch the dock at login, create `~/.config/autostart/godockelm.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=GoDockElm
Exec=/path/to/GoDockElm
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
```

## License

MIT
