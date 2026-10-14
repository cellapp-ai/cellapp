---
name: cellapp-logo
description: Reuse and refine CellApp brand assets, including square icons, landscape wordmarks, portrait lockups, light-background variants, PNG exports, and website integration. Use for CellApp logo requests and brand asset updates.
---

# cellapp (c//) Logo Design System & Prompt Guide

Current asset baseline: 2026-09-29. Reuse the vector assets for exact brand work; their geometry takes precedence over descriptive prompts.

## Core Brand Concept & Metaphor

- **Brand Name**: cellapp
- **Visual Elements**: Hexagon (`c`) + Double Forward Slash (`//`) derived from `Cll`.
- **Metaphor**:
  - **Hexagon (`c`)**: Represents a biological/digital "Cell" (foundational data/computing unit).
  - **Double Slash (`//`)**: Mimics `http://` to signify a **new AI-driven app protocol/schema**.

## Design Language Guidelines

| Dimension | Specification |
| :--- | :--- |
| **Geometry** | Hexagonal C outline and two parallel slashes; match the initial SVG geometry. |
| **Structure** | 'c' (Cell) + '//' (Protocol Invocation / Scheme). |
| **Edges & Corners** | Rigid multi-angle body paired with uniform **corner fillets/rounded ends**. |
| **Dynamics** | Rightward tilt to convey speed, progression, and active invocation. |
| **Color Scheme** | High-contrast Monochrome (Pure White on Dark/Black background). |

## Choose the asset

| Asset | Format / dimensions | Composition and background |
| --- | --- | --- |
| [cellapp-square.svg](assets/cellapp-square.svg) | SVG, 1024 × 1024; viewBox 0 0 100 100 | White geometric `c//` on opaque black |
| [cellapp-square-light.svg](assets/cellapp-square-light.svg) | SVG, 1024 × 1024; viewBox 0 0 100 100 | Black geometric `c//` on opaque white |
| [cellapp-square.png](assets/cellapp-square.png) | PNG, 1024 × 1024 | Raster export of the black-background square |
| [cellapp-square-light.png](assets/cellapp-square-light.png) | PNG, 1024 × 1024 | Raster export of the white-background square |
| [cellapp-landscape.svg](assets/cellapp-landscape.svg) | SVG, 493 × 100 | White outlined Michroma `ce//app`, transparent background |
| [cellapp-landscape-light.svg](assets/cellapp-landscape-light.svg) | SVG, 493 × 100 | Black outlined Michroma `ce//app`, transparent background |
| [cellapp-portrait.svg](assets/cellapp-portrait.svg) | SVG, 400 × 440 | White `c//` icon above `cellapp`, transparent background |
| [cellapp-portrait-light.svg](assets/cellapp-portrait-light.svg) | SVG, 400 × 440 | Black `c//` icon above `cellapp`, transparent background |

`-light` means **for a light background**, so the artwork is black. It does not mean white artwork. Unsuffixed landscape and portrait assets are for dark backgrounds; square assets supply their own opaque background.

The portrait wordmark reads **cellapp** with two upright lowercase `l` glyphs. Only its upper icon contains `//`. The landscape wordmark retains **ce//app**. Do not substitute one spelling or lockup for the other without a user request.

Use square for icons and favicons, landscape for navigation and horizontal branding, and portrait for vertically stacked branding. PNG copies support destinations that do not accept SVG.

## Source and snapshots

In this repository, `public/logos/` is the editable source of truth. The eight matching files in this skill's `assets/` are reusable snapshots and should match the public files byte for byte. Outside this repository, reuse the bundled assets without assuming the website paths exist.

Historical snapshots [assets/logo.svg](assets/logo.svg) and [assets/logo-horizontal.svg](assets/logo-horizontal.svg) are retained for reference to the 2026-09-28 versions. They are not current asset names. Preserve these historical files unless the user asks to replace or remove them; requested current updates apply to the `cellapp-*` assets.

The old website paths `public/logo.svg` and `public/logo-horizontal.svg` have moved to `public/logos/cellapp-square.svg` and `public/logos/cellapp-landscape.svg` respectively. Do not recreate the old paths during ordinary reuse.

## Geometry and typography

- Preserve the existing geometric C, two parallel forward slashes, corner fillets and proportions when reusing the mark. The square uses vector paths plus a reused slash, with no embedded raster image.
- Landscape uses the supplied Michroma outlines, preserving spacing and kerning. Portrait uses the same typeface with upright `ll`; its upper symbol reuses the square mark without the background rectangle.
- Both wordmarks are outlined SVGs and need no runtime font download. Retain the existing font attribution in their SVG descriptions.
- Keep each lockup's aspect ratio. Do not stretch, rotate, retype or rearrange it for ordinary placement. For requested refinements, edit a copy of the appropriate current SVG and change only the requested geometry, colors or layout.
- Use monochrome artwork and adequate background contrast. Gradients, textures, 3D rendering or altered glyphs require an explicit creative direction.

## Website integration

When updating this website's brand assets, read [the design specification](../../../docs/standards/design.md), particularly its brand assets section.

Synchronize the affected representations:

- `public/logos/cellapp-square.svg` supplies the `logo` Base64 data URI in `src/logo.ts`, used for the static page favicon.
- `public/logos/cellapp-landscape.svg` supplies the `wordmark` Base64 data URI in `src/logo.ts`, used in Header and Footer.
- `public/favicon.svg` and `public/icons.svg` are exact copies of the **unsuffixed square SVG**. Keep them in sync when that square changes.
- PNG files are rendered from the corresponding square SVG at 1024 × 1024. Regenerate affected PNGs instead of editing them independently.
- Update the corresponding current skill assets when public brand files change. Maintain the design specification's names, spelling, dimensions and background guidance.

`src/logo.ts` encodes the original SVG bytes, not a reconstructed or normalized SVG. Verify that decoding each URI yields the matching public source bytes.

The full build (`npm run build`, or `pnpm run build` when pnpm is available) embeds website assets, exports the pages, copies `public/logos/` to `dist/logos/`, and copies the root favicon / icons SVGs to `dist/`. Public logo URLs are `/logos/<filename>`; portable `dist/bundle.html` keeps its displayed branding inline. Do not change unrelated website layout or navigation just to update unused logo variants.

With librsvg available, regenerate square PNGs from the repository root:

```sh
rsvg-convert --width 1024 --height 1024 --output public/logos/cellapp-square.png public/logos/cellapp-square.svg
rsvg-convert --width 1024 --height 1024 --output public/logos/cellapp-square-light.png public/logos/cellapp-square-light.svg
```

Validate XML, dimensions, asset-copy equality and any affected embedded data. Render modified variants against their intended backgrounds and inspect wordmark spelling, alignment and clipping. For displayed branding or build changes, verify the generated website; snapshot-only skill updates require asset and skill validation rather than rebuilding unchanged pages.

## Core Prompts for AI Generators (Midjourney / SD / DALL-E)

Use these only for requested explorations or mockups. Exact logos should reuse the supplied vectors.

### Standard Vector / Flat App Icon

```text
Minimalist vector logo for "cellapp", AI protocol scheme "c//". The letter "c" is a sharp geometric hexagon representing a digital cell. Followed by two bold parallel slashes "//" matching the right angle of the hexagon. Represents a new URI protocol like http://. Clean, technical, flat monochrome white on black background, consistent corner fillets.
```

### Rigid Industrial / Tech Aesthetic

```text
Modern industrial tech logo for an AI app protocol "c//". Hexagonal skeleton forming the "c" cell, seamlessly followed by double oblique slashes "//". Slashes feature subtle rounded corners matching the hexagon's vertices. Forward-leaning symmetric tilt, high contrast black and white, precision vector icon, technical aesthetic.
```

### Portrait exploration

```text
CellApp portrait lockup: the supplied geometric c// icon centered above the outlined Michroma wordmark cellapp. Spell the lower wordmark with two upright lowercase l letters, not slashes. Transparent background, white artwork for dark surfaces or black artwork for light surfaces. Preserve the supplied geometry and spacing.
```

The symbol's cell / protocol metaphor can inform requested creative work; it does not authorize changes to the established artwork or product claims.

## Key Prompt Keywords (Triggers)

- **Form**: `hexagonal skeleton`, `parallel slashes`, `URI scheme abstraction`, `c//`
- **Style**: `minimalist vector`, `technical precision`, `corner fillets`, `forward tilt`
- **Negative**: `3D gradient`, `soft circle`, `sketch`, `complex texture`
