# DEMviewer (demoviewer)

Jump stats, 3D replay and what-if simulation for CS 1.6 kreedz demos (.dem).
By Claude & beggineR.

## Build

Needs Go 1.24 or newer. Everything else is in this folder, so no internet is needed.

```
sh build.sh
```

`build.sh` assembles `web/index.html` from `web/head.html`, `web/app.js` and
`web/renderer.js`, then builds:

- `dist/DEMviewer.exe`: Windows, its own window (WebView2), no console
- `dist/demviewer-linux`: Linux, opens in the browser

Run `build.sh` again after editing anything in `web/`, because the HTML is embedded into the binary.

Building by hand on Windows (PowerShell), after `web/index.html` exists:

```
$env:GOFLAGS="-mod=mod"; $env:GOPROXY="off"
go build -trimpath -ldflags "-s -w -H=windowsgui" -o DEMviewer.exe .
```

## Files

| File | What it does |
|---|---|
| `demo.go` | Reads the GoldSrc .dem format: header, directory, frames (origin, velocity, view, usercmd), text messages |
| `net.go` | Decodes network messages: delta descriptions, baselines and packet entities, used to place moving brush entities (bhop blocks, arena doors) |
| `bsp.go` | Loads maps (built-in `embedmaps/*.bsp.gz` first, then map folders), block edges and floors |
| `stats.go` | Jump detection and kz_jump_stats-style stats: distance, pre, jumpoff, block, landing, strafes, efficiency, GC efficiency, jump type, mouse slaps, replay data |
| `sim.go` | What-if simulation: re-runs the real post-jump inputs with a different pre, jumpoff or FOG |
| `main.go` | Local HTTP server and API, folder scanning, the list filter |
| `ui_windows.go`, `dialog_windows.go` | WebView2 window and native Windows browse dialogs |
| `ui_other.go`, `dialog_other.go` | Browser and zenity/kdialog on Linux |
| `settings.go` | Saves the demo and maps folder |
| `cvars.go` | League settings check (movevars, cl_lw, fps) |
| `check.go` | Dev tools: `-check <folder>` compares stats with the plugin output recorded in demos; `-whatif <dem> <pre> <jumpoff>` |
| `web/` | UI (`app.js`), WebGL BSP renderer (`renderer.js`), page and styles (`head.html`) |
| `rsrc_windows_amd64.syso` | Windows icon resource (from `logo/DEMviewer.ico`) |
| `third_party/` | Vendored go-webview2, go-winloader and golang.org/x/sys, each with its own license |
| `embedmaps/` | kz_longjumps2, kz_longjumps3 and kz_longjumparena, built into the exe |
