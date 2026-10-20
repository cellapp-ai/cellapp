# Proposal

## Why

The owner dashboard presents real management functions, but its wide-screen spacing, thin application rows, weak status hierarchy, and inconsistent page grouping make it feel like a prototype. Application details report only the data service state and version even though owners need to inspect the structure of tables already applied to their application.

## What Changes

- Refresh the full authenticated dashboard with a cohesive dark Cellapp interface: a compact navigation shell, clearer page hierarchy, readable application list and detail layouts, and consistent credential, device approval, and profile pages.
- Make real application state and available actions easy to scan without adding fabricated metrics, deployment history, or a browser deployment flow.
- Add an owner-only, read-only view of the latest applied application table structure: tables, columns, types, nullability, primary keys, indexes, and access policies. Do not show table rows or add table editing in the dashboard.
- Give loading, empty, unpublished, unavailable, and error states distinct presentation and recovery paths across the affected pages.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `web-control-plane`: strengthen the owner dashboard's usable application navigation and add authorized, read-only table structure inspection.

## Impact

- `apps/web` page composition, styles, client response validation, and browser acceptance checks.
- A backward-compatible owner-session JSON read under `/api/console/apps/{app}/data` (or a dedicated child read route) using the latest applied manifest in the separate application data database; update the Web API contract and server checks.
- No CLI or application data write contract change, no database migration, and no visitor access to schema metadata.
