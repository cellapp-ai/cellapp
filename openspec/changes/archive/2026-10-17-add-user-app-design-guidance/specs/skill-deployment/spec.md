# Spec Delta

## ADDED Requirements

### Requirement: Agent access to optional app-design guidance
The public Agent deployment instructions and deployment Skill SHALL direct an Agent to the user-app design guide when the request includes developing an app and lacks UI direction. They SHALL present it as adaptable guidance that does not require user opt-in or a particular framework or styling tool. A deployment-only request SHALL remain on the existing deployment path.

#### Scenario: Develop and deploy without a design
- **WHEN** a user asks an Agent to develop and deploy an app without supplying a UI design
- **THEN** the Agent-facing instructions direct it to the public design guide and allow it to continue without asking whether to use the guide

#### Scenario: Deploy an existing completed app
- **WHEN** a user asks an Agent only to publish an already built app
- **THEN** the deployment instructions do not require a redesign or a new framework choice before publication

### Requirement: Bilingual Agent entry points
The public Agent deployment runbook and packaged deployment Skill SHALL each include complete English and Simplified Chinese editions with reciprocal language links. The Chinese editions SHALL preserve the deployment, authorization, recovery, design-guidance, and capability boundaries of their English counterparts.

#### Scenario: Chinese-speaking Agent reads a public entry point
- **WHEN** an unauthenticated reader opens the Chinese deployment runbook or packaged Chinese Skill reference
- **THEN** the full workflow is available in Chinese and links back to the English edition
