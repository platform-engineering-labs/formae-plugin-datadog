# Datadog Plugin for Formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml)

Formae plugin for managing Datadog resources.

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `Datadog::Monitoring::Monitor` | Monitors (metric, query, composite alerts) |

## Installation

```bash
make install
```

## Configuration

Configure a Datadog target in your Forma file:

```pkl
import "@formae/formae.pkl"
import "@datadog/datadog.pkl"

new formae.Target {
    label = "datadog-target"
    namespace = "DATADOG"
    config = new datadog.Config {
        apiKey = read("env:DD_API_KEY")
        appKey = read("env:DD_APP_KEY")
        site = read("env:DD_SITE")
    }
}
```

Authentication uses Datadog API and Application keys:

```bash
export DD_API_KEY="your-api-key"
export DD_APP_KEY="your-application-key"
export DD_SITE="datadoghq.com"    # or us5.datadoghq.com, etc.
```

## Examples

See [examples/](examples/) for usage patterns:

- `basic/` - CPU usage monitor with warning and critical thresholds

## Development

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
```

## License

FSL-1.1-ALv2
