# DEMviewer

Jump stats, 3D replay, strafe and yaw analysis and what-if simulation for **CS 1.6 kreedz demos** (.dem).
By Claude & beggineR.

![Jump list](docs/list.png)

## Download

Get the latest **DEMviewer.exe** from [Releases](../../releases), then just run it: no install needed.
`kz_longjumps2`, `kz_longjumps3` and `kz_longjumparena` are built in. Other maps are picked up from your
`cstrike/maps` or `cstrike_downloads/maps` folder, or from a maps folder you choose in the app.

Linux: build it yourself (see Build below). It opens in your browser.

## What it does

**Jump list**
- Finds every jump in every demo in a folder: LJ, HJ, CJ, DCJ, MCJ, WJ, BJ, SBJ and ladder jumps, with backwards and sideways detection.
- **Best jumps** shows one valid landed jump per demo; **all attempts** shows every jump with the time it happens in the demo.
- Click any column header to sort; the type filter and the demo's file name (`_lj_`, `_sbj_`, `_ldj_` ...) help find the right jump.

**Stats that match the server plugins**

![Jump stats](docs/stats.png)

- Distance, prestrafe, FOG, jumpoff, block, landing / lasting, maxspeed and gain, strafes and efficiency, rebuilt to match
  **kz_jump_stats** (distance within about 0.02 on tested demos) and the [MAIN] ladder stats.
- Strafe table with gain, loss, eff, overlaps, **dead airtime** and GC-style turn efficiency.
- **Mouse slaps**: strafe switches where the mouse hits the other side like a wall (2-frame swing of 110% or more),
  calibrated on known slap and no-slap demos.
- **Yaw acceleration at switches**, average and peak.
- **Settings check**: server movement settings, cl_lw and fps from the demo, compared with the league rules.

**Timeline**

![Timeline](docs/timeline.png)

- Yaw speed and yaw acceleration graphs, and one block per frame for every key: jump, duck, forward, back, moveleft, moveright, ground, duck state and dead air.
- Mouse wheel zooms, dragging moves, clicking jumps the replay to that frame, hovering shows the frame (or the jump's stats).

**3D replay in the real map**

![Replay](docs/replay.jpg)

- First person, behind, side or top view with the player's hitbox, so edge landings are easy to see.
- Moving blocks (bhop blocks, longjumparena layouts) move like they did in the demo.

**What-if**

![What-if](docs/whatif.png)

- Re-runs the player's real inputs from take-off with a different prestrafe, jumpoff or FOG, using GoldSrc air physics.
- Shows whether it lands, the new distance, a self-check of the simulator against the real jump, and a replay with the real jump as a ghost.
  Simulated replays are always marked **WHAT-IF · SIMULATED**.

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
