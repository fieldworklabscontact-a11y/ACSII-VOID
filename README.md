# ASCII-VOID
Play in your browser: Launch ASCII-VOID: https://strong-croissant-e287e0.netlify.app
Arrow keys to move around.
**An experimental space game built with Go and Ebitengine.**

The project is being developed in Go using Ebitengine.

##Documentation:
My mathematical process can be found here: https://docs.google.com/document/d/1YPE8FrCMTuRr7MG2URQlLjU1UTmKCfeyCAXMB0OSDrA/edit?usp=sharing

## Current Development

ASCII-VOID is actively being developed. Current implemented features and experiments include:

- **Ebitengine integration:** A working game environment using Go and Ebitengine.
- **Movement system:** A velocity-based movement controller using cubic Bézier interpolation.
- **Inertial movement:** Adjustable transition timing to create a sense of mass and inertia.
- **Directional orientation:** The player representation rotates according to its total velocity vector.
- **Rendering experiments:** Screen resolution, pixel smoothing, and display behavior experimentation.
- **WebAssembly support:** A browser-compatible build that runs through Go's WebAssembly runtime.
-**Procedurally animated boss:** An environmental creature built from my own SVG art, with huge branches entering from off-screen and eyes at the screen edges. It is never fully visible.
-**Four eyes with distinct attacks:** The Executioner, Stalker, Divider, and Witness are always active, each with its own state machine, coordinated by a director that combines their attacks.
-**Boss phases:** A scripted phase sequence that controls which eyes and branches are active, how intense the attacks are, and when the screen goes dark.
-**Environmental manifestations:** Screen distortion, abrupt darkness, glitching, screen shake, and enormous shapes entering the viewport.
-**Hit feedback:** Just Shapes & Beats-style feedback on damage, with hitstop, screen flash, shake, shards, and HP pips.

## Technical Direction

Planned development areas include:

- Procedural and fractal-based geometry.
- Sierpiński triangle-based visual designs.
- An ASCII rendering pipeline.
- More advanced movement and physics systems.
- Procedural enemy behavior.
- Fractal-based kaiju encounters and combat.
- Expanded world and gameplay systems.

These are development goals, and will likely adjust or update them as I progress.

## Technology

- **Language:** Go
- **Game engine:** Ebitengine
- **Browser support:** WebAssembly

## Project Philosophy

ASCII-VOID is an exploration of how mathematics, procedural generation, and rendering techniques can come together to create an unusual game experience.

The goal is not simply to build another space game, but to experiment with how movement, geometry, and visual representation can be designed from first principles.

## Development Status

**Status:** In active development.

This repository documents an ongoing engineering and game development project. Features and systems are being implemented incrementally.
