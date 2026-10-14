# Design

The HTTP entrypoint already verifies the edge secret and validates the asserted original Host before selecting a control or visitor route. Downstream visitor handlers currently read `r.Host`, which is the Cloud Run origin under the Worker route. They must use `s.requestHost(r)` for the handoff label and same origin checks. The helper preserves the direct Host when edge mode is disabled or proof fails.

Keep validation at the entrypoint, including lookup of the assigned application hostname. The authorization endpoint continues to verify that its `host` parameter belongs to the given application. No broadening of accepted host labels or cookie scope is needed.

Verify a published application with an authenticated edge request: first navigation carries its browser hostname into `/auth/app`, ticket issuance and consumption return to that hostname, the page loads after login, same origin visitor writes pass their origin gate, cross origin writes fail, and forged edge assertions cannot route the application. Run existing server checks and the isolated PostgreSQL integration test when the local database is available.
