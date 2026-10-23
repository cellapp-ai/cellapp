# Proposal

## Why

The Home menu item repeats the brand link, and the navigation action currently says Dashboard even when a visitor needs to sign in first. The homepage should make the home destination clear through the wordmark and give signed-out visitors a clearer next step.

## What Changes

- Remove Home from the text navigation. Keep the wordmark as the home link and highlight its two slashes on the homepage.
- Show “Deploy for free” on the navigation action for signed-out visitors and “Dashboard” when the current browser session is authenticated.
- Keep both action states linked to the existing dashboard entry and preserve the static site's mobile and single-file navigation behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `product-site-surfaces`: Update shared navigation, current-page indication, and the signed-out dashboard entry label.

## Impact

The change affects the website's shared header, wordmark display, browser session check, browser acceptance coverage, and the product-site navigation spec. It uses the existing same-origin `/api/console/session` endpoint without changing its contract or the server.
