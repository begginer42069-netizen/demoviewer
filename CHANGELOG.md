# Changelog

All notable changes to DEMviewer. New entries go under **Unreleased**; the
release workflow turns that section into the next version's release notes.

## Unreleased

### Changed
- Releases now come with a written changelog (this file) instead of a list of commits.

## v1.2.0 (2026-10-26)

### Added
- **Real player models in the replay.** The player is drawn with their CT/T model from the demo (urban, gign, sas, gsg9, terror, leet, arctic, guerilla), animated like in the game: run, walk, crouch and jump for the legs, aiming up and down with the weapon for the upper body.
- **Weapons from the demo.** The player holds the weapon they had in the demo (knife, USP, scout ...).
- **Viewmodel** in first person, with a new *Viewmodel* option: *As in the demo* (hidden if it was hidden, e.g. after `/hide`), *Always show*, or *Hide*.
- **Models folder** setting. Models are read from your own `cstrike/models` folder, found automatically on the usual Steam paths, or picked with *Browse*. Nothing is bundled with DEMviewer, so you see your own (custom) models.
- **Player** option: model, hitbox, or both. The hitbox is still the best way to judge edge landings.
- **Fullscreen replay** with play/pause, *From take-off*, speed and a time slider on screen.
- **0.75x** replay speed.

### Changed
- The what-if ghost is now drawn as the opposite team's model (CT real run, T simulated run, and the other way round), so the two runs can't be mixed up.

## v1.1.0 (2026-10-25)

### Added
- Automatic Windows and Linux builds on GitHub, with semantic version numbers. The version is shown in the app header.
- The what-if replay now carries on after a simulated landing: the player lands and slides out on the block with ground friction.

## v1.0 (2026-10-25)

First public release.

- Jump list for a whole folder of demos: LJ, HJ, CJ, DCJ, MCJ, WJ, BJ, SBJ and ladder jumps, backwards and sideways detection, best jumps or all attempts with their time in the demo, sortable columns and type filter.
- Stats rebuilt to match kz_jump_stats and [MAIN]: distance, prestrafe, FOG, jumpoff, block, landing / lasting, maxspeed, strafes and efficiency, GC turn efficiency, dead airtime, mouse slaps, yaw acceleration at switches.
- League settings check (server movement settings, cl_lw, fps) read from the demo.
- Timeline with yaw speed, yaw acceleration and every key per frame.
- 3D replay in the real map with moving blocks, first person and third-person cameras with the hitbox.
- What-if simulation: the real inputs from take-off re-run with a different prestrafe, jumpoff or FOG.
- Built-in kz_longjumps2, kz_longjumps3 and kz_longjumparena.
