# systemd Deployment

These example units run Whooper as a local bridge service:

- `whooper-sync.service` keeps the SQLite cache fresh with `whooper sync --loop`.
- `whooper-serve.service` exposes `/healthz`, `/status`, `/metrics`, and `/api/*`.

The examples use `/var/lib/whooper` as `WHOOPER_HOME`. Run login once before
starting the services so that directory contains `config.yaml` and `token.json`.

```bash
sudo install -d -m 0700 /var/lib/whooper
sudo chown "$USER":"$USER" /var/lib/whooper

WHOOPER_HOME=/var/lib/whooper whooper config set client-id <client-id>
WHOOPER_HOME=/var/lib/whooper whooper config set client-secret <client-secret>
WHOOPER_HOME=/var/lib/whooper whooper login --no-browser
```

## User Service

Use user services when the data directory is owned by your user.

`~/.config/systemd/user/whooper-sync.service`:

```ini
[Unit]
Description=Whooper background sync
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=WHOOPER_HOME=/var/lib/whooper
ExecStart=/usr/local/bin/whooper sync --loop --interval 30m
Restart=always
RestartSec=30s

[Install]
WantedBy=default.target
```

`~/.config/systemd/user/whooper-serve.service`:

```ini
[Unit]
Description=Whooper metrics and HTTP API
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=WHOOPER_HOME=/var/lib/whooper
ExecStart=/usr/local/bin/whooper serve --addr 0.0.0.0:9464
Restart=always
RestartSec=30s

[Install]
WantedBy=default.target
```

Enable both:

```bash
systemctl --user daemon-reload
systemctl --user enable --now whooper-sync.service
systemctl --user enable --now whooper-serve.service
```

## System Service

For a system-level install, create a dedicated user and put the same units under
`/etc/systemd/system/`, adding `User=whooper` and `Group=whooper` to each
`[Service]` section.

```bash
sudo useradd --system --home /var/lib/whooper --shell /usr/sbin/nologin whooper
sudo install -d -o whooper -g whooper -m 0700 /var/lib/whooper
```

Run one-time setup as that user:

```bash
sudo -u whooper WHOOPER_HOME=/var/lib/whooper whooper config set client-id <client-id>
sudo -u whooper WHOOPER_HOME=/var/lib/whooper whooper config set client-secret <client-secret>
sudo -u whooper WHOOPER_HOME=/var/lib/whooper whooper login --no-browser
```

## Prometheus

Scrape the metrics service:

```yaml
scrape_configs:
  - job_name: whooper
    static_configs:
      - targets: ["whooper-host:9464"]
```

## Grafana

For SQLite dashboards, point Grafana's SQLite datasource at:

```text
/var/lib/whooper/whooper.db
```

For remote Grafana without sharing the SQLite file, use the JSON HTTP API
endpoints exposed by `whooper serve`.
