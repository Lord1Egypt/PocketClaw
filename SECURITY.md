# Security Policy

## Reporting a vulnerability

**Please do not open a public issue, pull request or discussion for a security
problem.**

Use GitHub's private reporting instead: open the repository's **Security** tab
and choose **Report a vulnerability**. That creates a private advisory visible
only to the maintainers. If private reporting is unavailable to you, contact the
repository owner directly through their GitHub profile and ask for a private
channel before sending any detail.

Please include what you need to make the issue reproducible: affected version
(`versionName` and `versionCode` from the app's About screen), device and
Android version, and the steps involved. A proof of concept helps, but a clear
description of the weakness is enough to start.

## What to expect

You will get an acknowledgement that the report was received, an assessment of
whether it is reproducible and what its impact is, and updates as a fix
progresses. Please give the maintainers a reasonable window to ship a fix before
disclosing publicly. Credit is offered unless you would rather stay anonymous.

## Scope

PocketClaw runs an agent runtime, a local gateway and real command-line tools on
the user's own device. The areas most worth your attention:

- Anything that lets another app on the device reach the gateway, its bearer
  credential, or the dashboard session.
- Anything that moves credentials or Core private state out of app-private,
  no-backup storage.
- Anything that lets untrusted input — a model response, a channel message, a
  workspace file — cause command execution or a path escape outside the
  workspace.
- Weaknesses in the packaged native payloads or in how they are verified.

Out of scope: the consequences of a user configuring their own device or
provider credentials insecurely, and issues in upstream dependencies that are
better reported to their own maintainers.

## Current status

This is pre-release software and is not yet production-signed. Please treat the
current builds accordingly.
