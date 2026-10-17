# Tasks

## 1. Publish the user-app guide

- [x] 1.1 Write `apps/website/public/deploy/app-design.md` with the fallback trigger, precedence for user and project design, visual and interaction guidance, responsive/accessibility checks, and framework-neutral static-build advice; verify the document against both delta specs and current Cellapp capabilities.
- [x] 1.2 Copy the guide into `apps/website/dist/deploy/` during website export and update the website README's output description; verify `corepack pnpm --dir apps/website run build` produces both public Markdown files.
- [x] 1.3 Extend the existing production-preview check to fetch the guide without authentication and verify the path and Markdown response; run the website lint, build, and preview test, recording any unavailable check accurately.

## 2. Connect the Agent entry points

- [x] 2.1 Update `apps/website/public/deploy/AGENTS.md` with a short conditional link for app development without UI direction and preserve the deploy-only route; verify the exported runbook and guide resolve together in the website preview.
- [x] 2.2 Update the public `skills/skills/cellapp-deploy/SKILL.md` with the same trigger, precedence, and guide URL; verify `npm --prefix skills run check:skill` and `npm --prefix skills run check:boundary`, plus a review that the Skill remains usable when the guide cannot be fetched.

## 3. Integrate and review

- [x] 3.1 Review both Agent entry points, the guide, and the website output for consistent branding, links, no framework mandate, no opt-in question, and no redesign requirement for deploy-only work; verify `openspec validate add-user-app-design-guidance --strict` and record the website and Skill release states separately.

## 4. Bilingual design guidance

- [x] 4.1 Add `apps/website/public/deploy/app-design.zh-CN.md` as a complete Simplified Chinese counterpart and cross-link both editions; verify section-by-section content and platform constraints match.
- [x] 4.2 Export both language files and update the website output documentation; verify the production build contains both Markdown editions.
- [x] 4.3 Extend the production-preview check to fetch both language paths and their cross-links; run the relevant website checks and record results.

## 5. Bilingual Agent entry points

- [x] 5.1 Add a complete `apps/website/public/deploy/AGENTS.zh-CN.md` and reciprocal links in both runbook editions; verify commands, URLs, authorization, recovery, and design-guidance trigger match the English edition.
- [x] 5.2 Export the Chinese runbook and update website output documentation and preview checks for both language URLs; verify website lint, build, and browser acceptance.
- [x] 5.3 Add a complete `skills/skills/cellapp-deploy/SKILL.zh-CN.md` and reciprocal links in both Skill editions; verify section-by-section parity and run `check:skill` and `check:boundary`.
- [x] 5.4 Recheck all three bilingual pairs, OpenSpec strict validation, and separate website and Skill release states; update the verification record.

## Workflow follow-up

- After implementation and required review, sync and archive the change under the project's OpenSpec workflow, then commit the affected repositories in submodule-first order.
- Publish the website guide before a public Skill release that depends on its URL; treat website upload and Skill publication as separate actions.
