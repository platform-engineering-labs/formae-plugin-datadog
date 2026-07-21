# Changelog

All notable changes to the formae Datadog plugin are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Install with `sudo formae plugin install datadog` on the host that runs the
formae agent.

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
