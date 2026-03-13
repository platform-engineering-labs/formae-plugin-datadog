# Basic Monitoring

Basic Datadog monitor with CPU usage alerting.

## What You Get

- Metric alert monitor on `system.cpu.user`
- Warning threshold at 80%, critical at 90%
- Renotify every 60 minutes

## Prerequisites

1. Datadog account with API access
2. Environment variables set: `DD_API_KEY`, `DD_APP_KEY`, `DD_SITE`

## Configuration

Edit `vars.pkl`:

```pkl
stackName = "datadog-monitoring"
```

Set your Datadog credentials:

```bash
export DD_API_KEY="your-api-key"
export DD_APP_KEY="your-app-key"
export DD_SITE="datadoghq.com"
```

## Deploy

```bash
formae apply --mode reconcile main.pkl
```

## Tear Down

```bash
formae destroy --query 'stack:datadog-monitoring'
```

## Architecture

```
Datadog Account
└── Monitor: High CPU Usage (metric alert)
```
