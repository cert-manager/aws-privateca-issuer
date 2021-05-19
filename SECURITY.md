# Vulnerability Reporting Process

Security is the number one priority for the AWS Private Certificate Authority (AWS PCA) external issuer for cert-manager. If you think you've found a
security vulnerability in the AWS PCA external issuer for 
cert-manager, you're in the right place.

Our reporting procedure is a work-in-progress, and will evolve over time. We
welcome advice, feedback and pull requests for improving our security
reporting processes.

## Covered Repositories and Issues

This reporting process is intended only for security issues in the AWS PCA external
issuer itself, and doesn't apply to applications _using_ the exteral issuer or to
issues which do not affect security.

Broadly speaking, if the issue cannot be fixed by a change to the AWS PCA external issuer
, then it might not be appropriate to use this reporting
mechanism and a GitHub issue in the appropriate repo.

All that said, **if you're unsure** please reach out using this process before
raising your issue through another channel. We'd rather err on the side of
caution!

## Reporting Process

1. Describe the issue in English, ideally with some example configuration or
   code which allows the issue to be reproduced. Explain why you believe this
   to be a security issue in AWS PCA external issuer, if that's not obvious.
2. Put that information into an email. Use a descriptive title.
3. Send the email to [`AWS Security and the Maintainers of this Plugin`](mailto:aws-security@amazon.com,setparam@amazon.com,baiakbar@amazon.com,kontakt@ju-hh.de)

## Response

Response times could be affected by weekends, holidays, breaks or time zone
differences. That said, the security response team will endeavour to reply as
soon as possible.

As soon as the team decides that the report is of a genuine vulnerability,
one of the team will respond to the reporter acknowledging the issue and
establishing a disclosure timeline, which should be as soon as possible.