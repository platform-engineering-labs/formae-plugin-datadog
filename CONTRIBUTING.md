# Contributing

This document covers local development for plugin authors. For user-facing
plugin docs (configuration, supported resources, examples), see
[README.md](README.md).

## Local Installation

```bash
make install
```

## Building & Testing

```bash
make build          # Build plugin
make test           # Run tests
make install        # Install locally
make gen-pkl        # Resolve PKL dependencies
```

## Conformance Tests

Run against a real Datadog account:

```bash
make conformance-test-crud TEST=monitor
make conformance-test-crud TEST=slo
make conformance-test-crud TEST=downtime-schedule
make conformance-test-crud TEST=index
make conformance-test-crud TEST=logs-metric
make conformance-test-crud TEST=role
make conformance-test-crud TEST=team
make conformance-test-crud TEST=dashboard
make conformance-test-crud TEST=synthetics-test
make conformance-test-crud TEST=pipeline
```
