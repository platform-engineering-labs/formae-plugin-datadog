# Datadog Plugin for Formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml)
[![Nightly](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/nightly.yml)

Formae plugin for managing Datadog resources.

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `Datadog::Monitoring::Monitor` | Monitors (metric, query, composite alerts) |
| `Datadog::Monitoring::SLO` | Service Level Objectives (metric, monitor types) |
| `Datadog::Monitoring::DowntimeSchedule` | Downtime schedules (one-time, recurring) |
| `Datadog::Logs::Index` | Logs indexes (filter, exclusion filters, retention) |
| `Datadog::Logs::Metric` | Log-based metrics (count, distribution aggregations) |
| `Datadog::Logs::Archive` | Logs archives (S3, GCS, Azure destinations) |
| `Datadog::IAM::Role` | Custom roles with permission management |
| `Datadog::IAM::Team` | Teams (name, handle, description) |
| `Datadog::Security::MonitoringRule` | Security monitoring detection rules |
| `Datadog::Dashboard::Dashboard` | Dashboards (discovery-first, raw JSON widgets) |
| `Datadog::Synthetics::Test` | Synthetics API tests (discovery-first, raw JSON config) |
| `Datadog::Logs::Pipeline` | Logs pipelines (typed fields + raw JSON processors) |

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

## License

FSL-1.1-ALv2
