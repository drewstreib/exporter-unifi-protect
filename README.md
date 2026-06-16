# UniFi Protect Exporter

[![lint](https://github.com/drewstreib/exporter-unifi-protect/actions/workflows/golangci.yml/badge.svg)](https://github.com/drewstreib/exporter-unifi-protect/actions/workflows/golangci.yml)
[![release](https://github.com/drewstreib/exporter-unifi-protect/actions/workflows/goreleaser.yml/badge.svg)](https://github.com/drewstreib/exporter-unifi-protect/actions/workflows/goreleaser.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE.md)

A [Prometheus](https://prometheus.io/) exporter for [UniFi Protect](https://ui.com/camera-security)
sensors. It authenticates against your UniFi console, polls sensor state on each scrape, and exposes
it as Prometheus metrics — temperature, humidity, light, air quality, battery, signal strength,
leak/motion/open detection, and device status.

## Features

- **Air quality** — full support for the UP Air Quality sensor (AQI, CO₂, PM1/2.5/4/10, VOC, TVOC, vape).
- **Unified metrics** — every sensor's temperature/humidity is exported under the same metric name,
  whether it comes from a USL Environmental or a UP Air Quality device.
- **No bogus zeros** — a sensor only emits the metrics it actually supports.
- **Env-var / `.env` configuration** — every flag has a `UNIFI_*` variable; a `.env` file is loaded
  automatically.
- **Self-contained** — a single static binary / `FROM scratch` image, no runtime dependencies.

## Prerequisites

- A UniFi console (Dream Machine / UNVR / Cloud Key) running Protect.
- A **local** UniFi account (username + password). Ubiquiti SSO / 2FA accounts won't work; create a
  dedicated local, read-only user.
- Docker, or Go 1.22+ to build from source.

## Quick start

### Docker

```shell
docker run -d --name unifi-protect-exporter -p 9090:9090 \
  -e UNIFI_HOST=https://192.168.1.1 \
  -e UNIFI_USERNAME=readonly-user \
  -e UNIFI_PASSWORD=secret \
  ghcr.io/drewstreib/exporter-unifi-protect:latest
```

Images are published to the [GitHub Container Registry](https://github.com/drewstreib/exporter-unifi-protect/pkgs/container/exporter-unifi-protect)
per release; replace `latest` with a specific version tag if you prefer to pin. A full
Prometheus + Grafana stack is provided in [`compose/compose.yaml`](./compose/compose.yaml).

### Binary

Download the latest release for your platform with the install script, or grab it from the
[releases page](https://github.com/drewstreib/exporter-unifi-protect/releases):

```shell
curl -sfL https://raw.githubusercontent.com/drewstreib/exporter-unifi-protect/main/install.sh | sh -s -- -b /usr/local/bin
exporter-unifi-protect serve --host https://192.168.1.1 --username readonly-user --password secret
```

## Configuration

Configuration comes from CLI flags or environment variables. The three required settings are
`UNIFI_HOST`, `UNIFI_USERNAME` and `UNIFI_PASSWORD`; every other flag has a matching
`UNIFI_<FLAG_NAME>` variable (dots and dashes become underscores), e.g. `--web.listen-addresses` →
`UNIFI_WEB_LISTEN_ADDRESSES`, `--min-detection-span` → `UNIFI_MIN_DETECTION_SPAN`. Run
`exporter-unifi-protect serve --help` to see the variable for each flag.

On startup the exporter loads a `.env` file from the working directory if present. Real environment
variables always take precedence, so shell exports and docker-compose `env_file` values are never
overridden. Copy the example and fill it in:

```shell
cp .env.example .env   # then edit UNIFI_HOST / UNIFI_USERNAME / UNIFI_PASSWORD
exporter-unifi-protect serve
```

Key flags (defaults in parentheses):

| Flag | Env | Description |
|------|-----|-------------|
| `--host` | `UNIFI_HOST` | UniFi console URL, e.g. `https://192.168.1.1` (required) |
| `--username` / `--password` | `UNIFI_USERNAME` / `UNIFI_PASSWORD` | Local account credentials (required) |
| `--timeout` (`5s`) | `UNIFI_TIMEOUT` | Max duration for collecting data per scrape |
| `--min-detection-span` (`1m`) | `UNIFI_MIN_DETECTION_SPAN` | Window during which motion/open reads as detected |
| `--labels` | `UNIFI_LABELS` | Extra labels by device ID, e.g. `<id>=room:kitchen` (repeatable) |
| `--web.listen-addresses` (`:9090`) | `UNIFI_WEB_LISTEN_ADDRESSES` | Address(es) to listen on |
| `--web.config.file` | `UNIFI_WEB_CONFIG_FILE` | [TLS / basic-auth config](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md) |

## Endpoints

| Path | Description |
|------|-------------|
| `/metrics` | Prometheus metrics |
| `/-/healthy` | Health check (always `200 Healthy`) |
| `/` | Landing page with links |

## Metrics

All sensor metrics are gauges labelled with `id` and `name`. `sensor_info` carries device metadata
(firmware, model, type, NVR MAC, …) as labels with a constant value of `1`.

- **Environmental** — `sensor_temperature_celsius`, `sensor_humidity_percentage`, `sensor_light_lux`
- **Air quality** (UP Air Quality) — `sensor_air_quality_aqi`, `sensor_air_quality_co2_ppm`,
  `sensor_air_quality_pm1p0`, `sensor_air_quality_pm2p5`, `sensor_air_quality_pm4p0`,
  `sensor_air_quality_pm10p0`, `sensor_air_quality_voc`, `sensor_air_quality_tvoc`,
  `sensor_air_quality_vape`
- **Power & connectivity** — `sensor_battery_status_percentage`, `sensor_bluetooth_signal_strength`,
  `sensor_bluetooth_signal_quality`, `sensor_is_connected`
- **Detection** — `sensor_is_motion_detected`, `sensor_is_opened` (both windowed by
  `--min-detection-span`), `sensor_leak_detected_at`, `sensor_external_leak_detected_at`
- **Status & lifecycle** — `sensor_is_adopted`, `sensor_is_updating`, `sensor_is_provisioned`,
  `sensor_is_ssh_enabled`, … and the timestamps `sensor_up_since_gauge`, `sensor_last_seen_gauge`,
  `sensor_connected_since_gauge`

Metrics are fetched fresh from the UniFi API on every scrape (pull model); `--timeout` bounds each
collection.

## Prometheus

Add the exporter to your `scrape_configs`:

```yaml
scrape_configs:
  - job_name: unifi-protect
    static_configs:
      - targets: ['<exporter-host>:9090']
```

## Grafana

Add Prometheus as a data source, then import the example dashboard from
[`grafana/single-sensor.json`](./grafana/single-sensor.json).

<p align="center">
  <img src="./grafana/single-sensor.png" alt="Single Sensor dashboard" width="480">
</p>

## Building from source

```shell
git clone https://github.com/drewstreib/exporter-unifi-protect.git
cd exporter-unifi-protect
go build -o exporter-unifi-protect ./cmd/exporter-unifi-protect
go test ./...
```

## Forked from

A fork of [merlindorin/exporter-unifi-protect](https://github.com/merlindorin/exporter-unifi-protect)
by Romain DARY, now maintained independently. Since the fork it has added UP Air Quality support,
fixed leak-sensor metrics, moved configuration to environment variables, and replaced the upstream
`go-shared` / `go-unifi-protect` dependencies with a self-contained client.

## Contributing

Issues and pull requests are welcome. For larger changes, please open an issue first to discuss the
approach.

## License

MIT — see [LICENSE.md](./LICENSE.md). The file retains the original copyright of Romain DARY (the
upstream author) alongside contributions to this fork.
