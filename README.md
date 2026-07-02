# Datadog Plugin for Formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/ci.yml)
[![Nightly](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-datadog/actions/workflows/nightly.yml)

Formae plugin for managing Datadog resources.

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `DATADOG::Monitoring::Monitor` | Monitors (metric, query, composite alerts) |
| `DATADOG::Monitoring::SLO` | Service Level Objectives (metric, monitor types) |
| `DATADOG::Monitoring::DowntimeSchedule` | Downtime schedules (one-time, recurring) |
| `DATADOG::Logs::Index` | Logs indexes (filter, exclusion filters, retention) |
| `DATADOG::Logs::Metric` | Log-based metrics (count, distribution aggregations) |
| `DATADOG::Logs::Archive` | Logs archives (S3, GCS, Azure destinations) |
| `DATADOG::IAM::Role` | Custom roles with permission management |
| `DATADOG::IAM::Team` | Teams (name, handle, description) |
| `DATADOG::Security::MonitoringRule` | Security monitoring detection rules |
| `DATADOG::Dashboard::Dashboard` | Dashboards (discovery-first, raw JSON widgets) |
| `DATADOG::Synthetics::Test` | Synthetics API tests (discovery-first, raw JSON config) |
| `DATADOG::Logs::Pipeline` | Logs pipelines (typed fields + raw JSON processors) |

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
