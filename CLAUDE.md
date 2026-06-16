# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

A Prometheus exporter for UniFi Protect. It authenticates against a UniFi Protect
(Dream Machine / UNVR) host, polls sensor data on each scrape, and exposes it as Prometheus
metrics over HTTP. The primary subcommand is `serve` (plus `version` and `licence`).

## Build & Run

```shell
# Build (the main package lives one level below the repo root)
go build -o exporter-unifi-protect ./cmd/exporter-unifi-protect

# Run (flags are --host/--username/--password, not --unifi-*)
./exporter-unifi-protect serve --help
./exporter-unifi-protect serve --host https://<console> --username <user> --password <pass>
```

Config is layered (later sources override earlier): a `.env` file in the working directory (loaded by
`cli.LoadDotEnv` in `main`, never overriding real env vars) → environment variables → YAML files
(`/etc/unifi-protect/config.yaml`, `~/.hoomy/unifi-protect.yaml`) → CLI flags. **Every** flag has an
env var: `kong.DefaultEnvars("UNIFI")` auto-derives `UNIFI_<FLAG_NAME>` (e.g.
`UNIFI_WEB_LISTEN_ADDRESSES`), and the required ones keep explicit names (`UNIFI_HOST`,
`UNIFI_USERNAME`, `UNIFI_PASSWORD`). Use a **local** UniFi account, not a Ubiquiti SSO account.

Metrics are served at `/metrics` (default listen `:9090`), with `/-/healthy` and an HTML status
page at the route prefix. TLS/basic-auth are configured via `--web.config.file` (prometheus
exporter-toolkit web config format).

## Lint, Test, Release

```shell
go build ./...            # build
go test ./...             # tests (hermetic, no network — see Testing)
golangci-lint run ./...   # lint — golangci-lint v2 (config .golangci.yaml is version: "2")
gofmt -l internal/ cmd/   # formatting check
```

Lint uses a strict golangci-lint **v2** ruleset. Two gosec rules are intentionally excluded in
`.golangci.yaml`: **G117** (exported `Password` fields are intentional credential flags) and **G704**
(SSRF taint — the request target is the operator-configured UniFi host by design). CI runs lint
(`.github/workflows/golangci.yml`, Go 1.23 + golangci-lint v2) and, on `v*` tags, a goreleaser
release (`.github/workflows/goreleaser.yml`) that cross-compiles linux/darwin × amd64/arm64 and
pushes `ghcr.io/drewstreib/exporter-unifi-protect:{<tag>,latest}` (single-arch linux/amd64 scratch
image; no signing).

> The repo still carries `.tk/` taskfiles and aqua/go-task scaffolding, but the key configs
> (`.golangci.yaml`, the two workflows, `.goreleaser.yaml`) have been **hand-edited and have diverged**
> from that generator. Do not re-run the `task *:boilerplate` / `*:ci` regeneration tasks — they will
> revert the hand edits (golangci-lint v2 migration, Go 1.23, cosign removal, the drewstreib rename).

### Testing

Tests live in `internal/` and are hermetic:

- `collector_test.go` — drives `collectSensor` over real captured payloads in `internal/testdata/`
  (e.g. `up-airquality.json`), asserting the right metrics appear and unsupported ones are absent.
- `cli/dotenv_test.go` — the `.env` loader (parsing, precedence, missing/malformed files).
- `upstream_test.go` — asserts the collector still declares every `sensor_*` family the original
  merlindorin v0.0.8 exporter exposed (fixture `testdata/upstream-v0.0.8.metrics`), guarding the
  refactor against silently dropping a metric.

## Architecture

The flow is small and linear — three pieces:

1. **`cmd/exporter-unifi-protect/main.go`** — wires the CLI with [kong](https://github.com/alecthomas/kong).
   The root `CMD` struct embeds `*cli.Commons` (the local `internal/cli` package, which provides the
   version/licence subcommands and logger setup) and registers the `Serve` command. Build-time vars
   (`version`, `commit`, `date`, `buildSource`, `license`) are injected via ldflags.

2. **`cmd/exporter-unifi-protect/commads/serve.go`** — (note the misspelled `commads` package dir).
   The `Serve` command holds all flags. `Run` constructs an `internal.Client` (the UniFi Protect
   API client), builds a `prometheus.Registry`, registers the custom collector plus standard
   Go/build-info collectors, and serves via the prometheus exporter-toolkit `web` package with
   graceful SIGTERM shutdown. It also handles external-URL / route-prefix logic mirroring
   Prometheus's own behavior (`cli.ComputeExternalURL`, `cli.NewHTTPLogger`).

3. **`internal/collector.go` + `internal/sensor.go` + `internal/client.go`** — the `Collector`
   implements `prometheus.Collector`. On each `Collect` (bounded by `--timeout`) it calls
   `client.ListSensors`, which GETs `/proxy/protect/api/sensors` and decodes it into the
   **exporter-local `Sensor` model** (`sensor.go`). We model it locally so we control the schema —
   the field set covers the `airQuality` block and types many readings as nullable pointers (see
   presence-gating below). `client.go` is a self-contained UniFi client: login → cache the `TOKEN`
   cookie (a JWT, read only for its `exp`) + `X-CSRF-Token` → replay both, re-login on expiry.
   Per sensor the collector emits environment readings, battery, bluetooth, air-quality readings,
   boolean device-status gauges, and timestamp gauges.

### Key behaviors to know

- **Pull model**: metrics are fetched fresh from the UniFi API on every Prometheus scrape, not on
  a background interval. `--timeout` bounds each collection.
- **Detection windowing**: `sensor_is_motion_detected` and `sensor_is_opened` are derived by
  comparing `MotionDetectedAt` / `OpenStatusChangedAt` (UniFi timestamps are **microseconds**, see
  the `microsec` constant) against `--min-detection-span` (default `1m`). The metric reads `1` for
  the span after a detection event. The span (in seconds) is exposed as a `detected_period` label.
- **Presence-gating (device-type awareness)**: many `Sensor` fields are `null` on devices that
  don't support them (e.g. the UP Air Quality sensor reports `null` for `stats.*`, battery, and
  bluetooth, and instead populates `airQuality`). These fields are modeled as **pointers** and the
  collector skips a metric when its value is `nil` — so a sensor only emits the metrics it actually
  supports, instead of exporting a misleading `0`. The `measure()` helper enforces this for any
  `{value, status}` reading.
- **Unified temp/humidity**: the UP Air Quality sensor reports temperature/humidity inside its
  `airQuality` block (`stats.*` is null), but the collector surfaces them on the **common**
  `sensor_temperature_celsius` / `sensor_humidity_percentage` (falling back to `airQuality` when
  `stats` is absent), so every sensor is queryable uniformly. The air-quality-only readings (aqi,
  co2, pm1p0/2p5/4p0/10p0, voc, tvoc, vape) use `sensor_air_quality_*`.
- **Error handling**: the collector is constructed with `reportErrors=true`; an API failure emits an
  invalid metric rather than silently dropping data.
- Adding a new metric means: add a `*prometheus.Desc` field, init it in `NewCollector`, send it in
  `Describe`, and emit it in `collectSensor` from the corresponding `Sensor` field (gated on
  presence if the field is a pointer/optional).

## Dependencies

This project is self-contained: the UniFi Protect client (`internal/client.go`) and CLI/web
plumbing (`internal/cli`) are implemented in-repo on top of the standard library, so there are **no
first-party `merlindorin/*` dependencies** (they were removed — see git history). Direct deps are
all widely-used libraries: `alecthomas/kong` (+`kong-yaml`) for the CLI, `prometheus/client_golang`
+ `prometheus/exporter-toolkit` for metrics/serving, `go.uber.org/zap` for logging, and
`golang-jwt/jwt/v5` (used only to read the session token's expiry). Deps are vendored under
`vendor/`; run `go mod tidy && go mod vendor` after changing them.

## Deployment

- `Dockerfile` builds a `FROM scratch` image with default `CMD ["serve"]`; releases publish it to
  `ghcr.io/drewstreib/exporter-unifi-protect` (`:<tag>` and `:latest`).
- `compose/compose.yaml` runs the exporter (listening on `:9090`, scraped by Prometheus at
  `unifi-protect:9090`) alongside Prometheus + Grafana; it reads `../.env`.
- `helm/exporter-unifi-protect/` is a Helm chart.
- `grafana/single-sensor.json` is an importable example dashboard.
