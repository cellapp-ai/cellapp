# Spec Delta

## MODIFIED Requirements

### Requirement: Personal profile settings

Web SHALL provide the current signed-in account's Profile page, showing its full immutable account ID and sign-in mode, and allow setting or clearing a display name. For a GitHub account with no chosen display name, the visible name SHALL default to its latest verified GitHub username. Profile reads and updates MUST derive the account from the browser session, never a client-provided account ID; changing a display name MUST NOT change account identity, application addresses, or access.

#### Scenario: Open profile and save a name
- **WHEN** a signed-in user opens `/profile` and submits a valid display name
- **THEN** the page shows the real account information, the server changes only that account's chosen name, and a refreshed page still shows the saved value

#### Scenario: Clear or reject a name
- **WHEN** the user submits an empty string, or submits more than 80 characters, edge whitespace, control characters, or invalid JSON
- **THEN** an empty string clears the chosen name and reveals the GitHub username default when available; invalid input returns a clear error without changing the saved name

#### Scenario: Authentication and origin boundaries
- **WHEN** a request without a session or from a foreign origin tries to update profile information
- **THEN** it is rejected without changing account data; an existing CLI Bearer credential cannot replace the browser session

#### Scenario: Refresh and uncertain result
- **WHEN** the user refreshes the `/profile` deep link, or an update request fails with an uncertain result
- **THEN** the valid deep link loads; the UI does not report success or retry the write automatically and offers a fresh read

## ADDED Requirements

### Requirement: Verified GitHub account presentation

The authenticated dashboard SHALL show a GitHub owner's latest verified username by default and an avatar tied to the immutable GitHub user ID. It SHALL keep the name readable when the image is unavailable and prefer an explicitly chosen display name.

#### Scenario: Signed-in identity
- **WHEN** a GitHub owner signs in and opens the console
- **THEN** the account area shows the verified username and avatar without replacing the immutable account ID used for authorization

#### Scenario: Missing avatar or chosen name
- **WHEN** the GitHub avatar cannot load, or the owner chooses a display name
- **THEN** the account area remains readable and shows the chosen display name when present
