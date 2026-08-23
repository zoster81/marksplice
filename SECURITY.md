# Security Policy

Marksplice is currently a pre-v1 beta. Security fixes are provided for the latest published beta line only; older beta releases may require upgrading to receive a fix.

## Reporting a vulnerability

Do not disclose a suspected vulnerability in a public issue, discussion, pull request, or other public channel.

When the public GitHub repository has private vulnerability reporting enabled, use the repository's **Security** tab to submit a private report. Include the affected Marksplice version, a minimal reproduction, impact, and any relevant environment details.

If private vulnerability reporting is temporarily unavailable, contact the maintainer through the GitHub account associated with this repository and request a private reporting channel without including vulnerability details in the public message.

## Response and disclosure

Reports will be evaluated against the currently supported beta. Confirmed issues will be fixed in a new immutable module version; published Go module tags are never rewritten. Coordinated disclosure should wait until a fixed version is available unless there is a compelling safety reason to disclose sooner.
