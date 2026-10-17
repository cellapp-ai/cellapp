# Design

## Context

See `proposal.md` for motivation and the two delta specs for expected behavior. The public `/deploy/AGENTS.md` is a short Markdown file in `apps/website/public/deploy/`; the website export script copies it explicitly into `dist/deploy/`, and the website test fetches that production artifact. The public `skills/skills/cellapp-deploy/SKILL.md` is in a separate Git submodule. The console and website design standards describe Cellapp-owned surfaces and do not apply to user apps.

## Goals / Non-Goals

**Goals:**
- Give an Agent a brief, usable fallback when the user requests app development without a UI design, with no additional user-facing choice.
- Let the Agent choose or retain the app's framework and styling approach while making the resulting interface reviewable.
- Keep one public canonical guide and small pointers from existing Agent entry points.

**Non-Goals:**
- Create a UI kit, starter template, fixed palette, or mandatory scoring system for user apps.
- Change the console or website's own visual standards, the CLI, or static hosting behavior.
- Require design work when the user asks only to deploy an existing app.

## Decisions

### Publish a separate Markdown guide under `/deploy/`

Add `apps/website/public/deploy/app-design.md` and copy it to `dist/deploy/app-design.md` in the website's existing export script. This keeps the public URL adjacent to the runbook that Agents already read, while preventing the deployment checklist from becoming a long design manual. The `/deploy/` page and copyable prompt can continue to point to `AGENTS.md`; that file links to the new guide. The deployment Skill links to the same public URL. Alternative considered: duplicate full guidance in both Agent instructions and Skill; that would drift across the website and submodule release paths.

### Keep the design guide in two synchronized languages

Publish the original English `app-design.md` and a Simplified Chinese `app-design.zh-CN.md` beside it under `/deploy/`. Give each file a language switch link at the top and export both as Markdown. Keep the Agent entry points on the English URL as a stable default; readers can follow the Chinese link. Compare headings, recommendations, and platform limits during review. Alternative considered: one mixed-language file; that makes Agent reading and updates harder and does not give each language a stable direct URL.

### Pair the other public Agent documents

Add `AGENTS.zh-CN.md` beside `/deploy/AGENTS.md` and `SKILL.zh-CN.md` beside the canonical deployment Skill. Keep `AGENTS.md` and `SKILL.md` as the existing machine entry points, add reciprocal language links, and preserve complete command examples, URLs, security limits, and recovery steps in translation. Export both runbook editions; the Skill package already includes its directory. Test both public runbook URLs in the static preview and inspect both Skill editions for parity. Alternative considered: a mixed-language machine entry point, which would make the instructions longer and harder to scan.

### Trigger on missing UI direction, without an opt-in prompt

The two Agent entry points give a short decision rule: apply the design guide when asked to develop or substantially redesign an app and no UI direction is supplied; follow explicit design instructions and established project patterns otherwise. For deploy-only work, continue the existing publishing flow. The guide can explain that an Agent may ask only about genuinely missing product requirements, not whether to use the design guidance. Alternative considered: ask the user to choose a design mode or approve the guide, which adds the decision burden the user wants to remove.

### Specify outcomes and checks, not tools

Organize the guide around purpose and audience, visual direction, a compact reusable system, page and state coverage, responsive and accessible behavior, and a final review of the built artifact at narrow and wide widths. Use specific examples and a short checklist, but no required framework, CSS library, component kit, color scheme, or fixed breakpoints. State the Cellapp constraint separately: the output must be a genuine static frontend; build tooling may be local. Alternative considered: prescribe Tailwind for HTML or StyleX for JavaScript projects; that would turn a design guide into a framework decision policy and impose avoidable migration work.

### Verify published output and links

Extend the existing website production-preview test to fetch the new Markdown, check its content type and key guidance, and confirm the public deployment instructions link to it. Check the Skill's link and guidance through its existing skill/boundary checks plus a direct text review. This tests the actual static export and discoverability without adding a test that merely mirrors the guide's wording. Check the guide's advice for consistency with the platform's real static/data capabilities.

## Risks / Trade-offs

- [The website guide and Skill can publish at different times] -> Publish and verify the website guide before shipping a Skill that links to it; report each release separately.
- [External Skill users may be offline or the website may be unavailable] -> Keep the Skill's trigger and core precedence rule self-contained; treat the remote guide as additional advice, not a blocker to ordinary development or deployment.
- [A generic aesthetic recipe could make unrelated apps look alike] -> Require the Agent to derive direction from the user's app and content, and to inspect the existing project before applying defaults.
- [A checklist can imply automated accessibility certification] -> Require observable checks and honest reporting, without claiming full certification from a single tool or viewport.

## Migration Plan

No data or API migration. During implementation, build and preview the website's new guide before updating or publishing the public Skill pointer. The website upload and Skill/public repository release remain separate. Rollback can remove the link from Agent entry points while leaving the guide available; an already published guide does not affect hosted user apps.
