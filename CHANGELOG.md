# Changelog

All notable changes to the formae Datadog plugin are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Install with `sudo formae plugin install datadog` on the host that runs the
formae agent.

## [0.1.2]

### Added

- The API and application keys accept a resolvable, so they can be sourced from
  a formae-managed secret. The agent resolves them live before every call, so
  onboarding an account or rotating a key needs no agent restart, and the key is
  stored as a reference rather than sitting in the target config.
- Both keys now fall back to `DD_API_KEY` and `DD_APP_KEY` at call time, which
  the plugin previously did not do. Each falls back independently, so one key
  may come from a secret while the other stays in the environment.
- A missing key is now reported when the client is built, naming both the target
  config field and the environment variable, instead of surfacing later as an
  unexplained authorization failure from Datadog.

### Changed

- `apiKey` and `appKey` are optional, since either can now come from the
  environment, and both are declared mutable so rotating a key updates the
  target rather than replacing it.
- Requires formae 0.89.0 or later, which resolves references in target config.

## [0.1.1]

### Changed

- Updated the plugin to the latest hub and conformance test standards.
- Resource types now use the `DATADOG::` prefix (forma files that import the
  Datadog schema need no change).
- The PKL schema is published under its hub URI.

## [0.1.0]

### Added

- Initial release of the Datadog plugin as a standalone package built on the
  formae Plugin SDK.
