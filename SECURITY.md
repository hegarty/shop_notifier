# Security Policy

This service delivers business sales data to real notification destinations (phone
numbers, Slack webhooks, email addresses). Please report suspected vulnerabilities
privately: GitHub's private vulnerability reporting for this repository (Security tab ->
"Report a vulnerability"), or me@terencehegarty.com.

## Scope

Particularly interested in: anything that could leak one tenant's notification content or
destination to another tenant, and injection via `notification_channels.configuration`
(parsed as fixed-schema JSON per channel type, never interpolated into a command or
template engine).

## Secrets handling

No database credential or (once a real SMS vendor is added) provider API key should ever
be committed here — both are read from AWS Secrets Manager at runtime. Report any
committed secret via the private channel above immediately.
