# Prometheus Alert Examples for Whooper

These alert rules are designed for use with a Prometheus instance scraping the
`whooper service` or `whooper serve` `/metrics` endpoint.

> [!NOTE]
> While `whooper sync --loop` performs periodic synchronization, it does not
> expose an HTTP server. To scrape metrics during a sync loop, use the
> combined `whooper service` command or run `whooper serve` in parallel.

## Sync Staleness

Alert when any entity has not been successfully synced for more than 24 hours.

```yaml
groups:
  - name: whooper_sync
    rules:
      - alert: WhooperSyncStale
        expr: whooper_sync_stale == 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Whooper sync is stale for {{ $labels.entity }}"
          description: "Whooper has not successfully synced {{ $labels.entity }} data for over 24 hours."
```

## Infrastructure Issues

Alert when critical configuration is missing or the database cannot be opened.

```yaml
groups:
  - name: whooper_health
    rules:
      - alert: WhooperDatabaseOffline
        expr: whooper_db_open == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Whooper database is offline"
          description: "The Whooper SQLite database could not be opened."

      - alert: WhooperTokenMissing
        expr: whooper_token_present == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Whooper OAuth token is missing"
          description: "A valid OAuth token is not present. Run `whooper login` to authenticate."

      - alert: WhooperStatusErrors
        expr: whooper_status_errors_total > 0
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Whooper reporting errors"
          description: "The Whooper status check is reporting {{ $value }} errors."
```

## Health Alerts

Alert based on configured thresholds for recovery and strain.

```yaml
groups:
  - name: whooper_health_alerts
    rules:
      - alert: WhooperLowRecovery
        expr: whooper_alert_state{alert="low_recovery"} == 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Whooper: Low Recovery Alert"
          description: "Current recovery is below the configured threshold."

      - alert: WhooperHighStrain
        expr: whooper_alert_state{alert="high_strain"} == 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Whooper: High Strain Alert"
          description: "Current strain is above the configured threshold."
```

## Recording Rules (Optional)

You can use recording rules to simplify queries for the latest health metrics.

```yaml
groups:
  - name: whooper_metrics
    rules:
      - record: whooper:latest_recovery_score
        expr: whooper_latest_health_metric{metric="recovery_score"}
      - record: whooper:latest_day_strain
        expr: whooper_latest_health_metric{metric="day_strain"}
```
