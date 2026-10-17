# Design guidance for user apps on Cellapp

[简体中文版](./app-design.zh-CN.md)

Use this guide when a user asks you to develop or substantially redesign an app and has not supplied a UI design. Apply it as a set of suggestions without asking whether to enable it. Follow the user's visual requirements and the project's established design system when either exists. If product behavior is genuinely unclear, ask about that behavior; do not make the user choose a frontend framework or styling tool just to begin.

This guide is for apps owned by Cellapp users. It does not define the appearance of the Cellapp console or website. It is not a fixed theme or a component kit.

## Find the app's direction

Before drawing the interface, inspect the project and identify the app's audience, main task, content, and tone. Pick a visual direction that helps that particular task: a focused editor, a readable record, a quick utility, and a data-heavy workspace call for different layouts. Use real domain language and representative content. When the brief is sparse, make reasonable design decisions and keep them easy to revise.

Give the interface one clear visual idea. Establish a small set of reusable choices for color roles, type hierarchy, spacing, surfaces, and controls. Keep text readable and the primary action easy to find. Use borders, cards, images, and motion to communicate structure or state, not as automatic decoration. Avoid filling every area with identical cards or importing Cellapp's brand colors into a user's app without a reason.

## Design the whole task

Organize each screen around what the user needs to understand and do next. Use plain labels and action text that names the result. Keep navigation, forms, and repeated controls consistent. For data-dependent views, consider loading, empty, error, success, and disabled states where they can actually occur. Explain recovery after a failure. Do not show invented records or a false success state as if the product were working.

Choose the framework and styling approach from the app's needs and existing project. Preserve a working stack unless changing it solves a concrete problem. A library can help maintain styles; it does not choose a good layout or verify the interface for you.

## Make it usable across screens and input methods

Let content determine layout changes. Check a narrow phone viewport and a wide desktop viewport, then inspect widths where the layout changes. Important actions and information must remain available without page-wide horizontal scrolling. Long text and links should wrap; a wide table or code block may scroll inside its own region. Check enlarged text and long or translated content when relevant.

Use semantic headings, links, buttons, labels, and status messages. Ensure the main flow works with keyboard and touch, has visible focus, and does not depend on hover alone. Check text and control contrast, image alternatives, and reduced-motion preferences. Treat automated accessibility checks as one input; inspect focus order and the real interface yourself.

## Review the built result

Build the actual static output and open it in a browser. Review at least one narrow and one wide layout, including the primary flow and its meaningful failure or empty state. Look at screenshots to catch cramped spacing, weak hierarchy, clipping, inconsistent controls, and unintended overflow. Fix issues you find, then report what you checked and any remaining gaps; do not claim full accessibility certification from a checklist.

Cellapp publishes static frontend files built locally. Choose any suitable framework or no framework, but deliver a genuine static output with its required CSS, scripts, fonts, and assets. Cellapp does not run an app server or private backend. Keep private credentials out of bundled files; visitors who can access the app can read its frontend assets. If the app needs persistent shared data, follow the supported Cellapp application-table workflow separately from the visual design work.
