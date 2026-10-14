# Public release review

On 2026-09-28, the product owner approved a local preview and explicitly requested that pricing remain labeled as a draft. The following product details still need confirmation before their preview wording is removed:

| Item | Current preview | Product detail to confirm |
| --- | --- | --- |
| Starter | $0, one app, community support | Confirm name, free availability and limits |
| Ship | $19 monthly or $180 annually, ten apps | Confirm price, billing terms and access-key availability |
| Scale | $59 monthly or $588 annually, unlimited apps | Confirm price, support and usage limits |
| Plan actions | Open Docs with plan/cadence context | No checkout is implemented; confirm the eventual conversion destination |
| Product/installation destination | https://github.com/cellapp-ai, inherited from the original page | Confirm the public repository and installation instructions; external availability has not been verified |
| Documentation | Static-build preparation, deploy example, verification, security guidance and troubleshooting | Confirm current CLI command/help syntax and supply versioned API docs when available |
| Blog | Three representative static articles, each with a distinct full body and date | Confirm editorial copy and dates before public launch |
| Logo | Supplied hexagonal C, version 4 | Confirm this asset is the chosen brand variant |

All commercial values are visibly labeled as a preview. The September preview involved no payment, signup, publication or message to another person. The owner's approval completed task 5.4 for that preview.

On 2026-10-14, the owner approved publishing the current page content, including its explicitly labeled draft pricing and the remaining content described above. The site was then published to Cloudflare Pages at `https://cellapp.ai`. This approval applies to the current release; the listed product details still require confirmation before removing their draft or preview labels or making stronger product claims.

Later on 2026-10-14, the website source was merged into `apps/website` in the Cellapp repository. The deployment documentation was updated to the verified public `@cellapp/cli@0.2.1` command, the client repository, `/deploy/` and `/deploy/AGENTS.md`. This source merge does not itself upload a new Cloudflare Pages release. Pricing and paid sales remain previews.
