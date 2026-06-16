//nolint:testpackage // white-box test: exercises unexported collectSensor without a network round-trip
package internal

import (
	"encoding/json"
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// epsilon is the tolerance used when comparing gauge values.
const epsilon = 1e-6

// collect drives collectSensor for a single sensor and returns the gauge value
// of every metric it emitted, keyed by fully-qualified metric name.
func collect(t *testing.T, c *Collector, s *Sensor) map[string]float64 {
	t.Helper()

	// fqNameRe extracts the fully-qualified metric name from Desc.String().
	fqNameRe := regexp.MustCompile(`fqName: "([^"]+)"`)

	const chanBuffer = 256

	ch := make(chan prometheus.Metric, chanBuffer)
	c.collectSensor(ch, s)
	close(ch)

	out := make(map[string]float64)
	for m := range ch {
		var dm dto.Metric
		if err := m.Write(&dm); err != nil {
			t.Fatalf("write metric: %v", err)
		}

		match := fqNameRe.FindStringSubmatch(m.Desc().String())
		if match == nil {
			t.Fatalf("cannot parse fqName from %q", m.Desc().String())
		}

		out[match[1]] = dm.GetGauge().GetValue()
	}

	return out
}

func mustDecode(t *testing.T, raw string) *Sensor {
	t.Helper()

	var s Sensor
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("decode sensor: %v", err)
	}

	return &s
}

func assertValue(t *testing.T, got map[string]float64, name string, want float64) {
	t.Helper()

	v, ok := got[name]
	if !ok {
		t.Errorf("expected metric %q to be present", name)
		return
	}

	if math.Abs(v-want) > epsilon {
		t.Errorf("metric %q = %v, want %v", name, v, want)
	}
}

func assertAbsent(t *testing.T, got map[string]float64, name string) {
	t.Helper()

	if v, ok := got[name]; ok {
		t.Errorf("expected metric %q to be absent, got %v", name, v)
	}
}

// airQualityJSON is the real payload reported by a UP Air Quality sensor.
const airQualityJSON = `{
  "type": "UP-AirQuality",
  "name": "up-aq-back",
  "firmwareVersion": "1.0.12",
  "hardwareRevision": null,
  "nvrMac": "E438832F5900",
  "id": "6a31a74400183e03e4106cd4",
  "marketName": "UP Air Quality",
  "modelKey": "sensor",
  "isConnected": true,
  "isAdopted": true,
  "upSince": 1781640895033,
  "lastSeen": 1781649480033,
  "connectedSince": 1781641041327,
  "leakDetectedAt": null,
  "externalLeakDetectedAt": null,
  "isOpened": null,
  "openStatusChangedAt": null,
  "motionDetectedAt": null,
  "stats": {
    "light": { "value": null, "status": "unknown" },
    "humidity": { "value": null, "status": "unknown" },
    "temperature": { "value": null, "status": "unknown" }
  },
  "airQuality": {
    "aqi": { "value": 7, "status": "neutral" },
    "vape": { "value": 0, "status": "safe" },
    "tvoc": { "value": 5.8, "status": "neutral" },
    "pm1p0": { "value": 0.8, "status": "neutral" },
    "pm2p5": { "value": 1.79, "status": "neutral" },
    "pm4p0": { "value": 2.5, "status": "neutral" },
    "pm10p0": { "value": 2.9, "status": "neutral" },
    "humidity": { "value": 51, "status": "neutral" },
    "temperature": { "value": 25.4, "status": "neutral" },
    "voc": { "value": 67, "status": "neutral" },
    "co2": { "value": 452, "status": "neutral" }
  },
  "bluetoothConnectionState": null,
  "batteryStatus": { "percentage": null, "isLow": false }
}`

func TestCollectAirQualitySensor(t *testing.T) {
	c := NewCollector(nil, time.Minute, time.Second, true)
	got := collect(t, c, mustDecode(t, airQualityJSON))

	// Air-quality readings are exported.
	assertValue(t, got, "sensor_air_quality_aqi", 7)
	assertValue(t, got, "sensor_air_quality_tvoc", 5.8)
	assertValue(t, got, "sensor_air_quality_voc", 67)
	assertValue(t, got, "sensor_air_quality_co2_ppm", 452)
	assertValue(t, got, "sensor_air_quality_pm1p0", 0.8)
	assertValue(t, got, "sensor_air_quality_pm2p5", 1.79)
	assertValue(t, got, "sensor_air_quality_pm4p0", 2.5)
	assertValue(t, got, "sensor_air_quality_pm10p0", 2.9)
	assertValue(t, got, "sensor_air_quality_humidity_percentage", 51)
	assertValue(t, got, "sensor_air_quality_temperature_celsius", 25.4)

	// Generic device metrics still come through.
	assertValue(t, got, "sensor_info", 1)
	assertValue(t, got, "sensor_is_connected", 1)

	// Environmental/battery/bluetooth readings are null on this device and must
	// not be exported as a misleading zero.
	assertAbsent(t, got, "sensor_temperature_celsius")
	assertAbsent(t, got, "sensor_humidity_percentage")
	assertAbsent(t, got, "sensor_light_lux")
	assertAbsent(t, got, "sensor_battery_status_percentage")
	assertAbsent(t, got, "sensor_bluetooth_signal_quality")
	assertAbsent(t, got, "sensor_bluetooth_signal_strength")

	// Detection metrics are skipped when the device does not report them.
	assertAbsent(t, got, "sensor_is_motion_detected")
	assertAbsent(t, got, "sensor_is_opened")
	assertAbsent(t, got, "sensor_leak_detected_at")
	assertAbsent(t, got, "sensor_external_leak_detected_at")
}

// environmentalJSON is a USL Environmental sensor: it reports stats/battery/
// bluetooth but no air quality.
const environmentalJSON = `{
  "type": "USL-Environmental-US",
  "name": "usl-foyer",
  "id": "6a2cb02a034a3e03e4034487",
  "marketName": "USL Environmental",
  "modelKey": "sensor",
  "isConnected": true,
  "stats": {
    "light": { "value": 23, "status": "neutral" },
    "humidity": { "value": 47, "status": "neutral" },
    "temperature": { "value": 26.11, "status": "neutral" }
  },
  "airQuality": null,
  "bluetoothConnectionState": { "signalQuality": 72, "signalStrength": -74 },
  "batteryStatus": { "percentage": 100, "isLow": false }
}`

func TestCollectEnvironmentalSensor(t *testing.T) {
	c := NewCollector(nil, time.Minute, time.Second, true)
	got := collect(t, c, mustDecode(t, environmentalJSON))

	// Environmental readings are exported.
	assertValue(t, got, "sensor_temperature_celsius", 26.11)
	assertValue(t, got, "sensor_humidity_percentage", 47)
	assertValue(t, got, "sensor_light_lux", 23)
	assertValue(t, got, "sensor_battery_status_percentage", 100)
	assertValue(t, got, "sensor_bluetooth_signal_quality", 72)
	assertValue(t, got, "sensor_bluetooth_signal_strength", -74)

	// No air-quality metrics for a device that does not report them.
	assertAbsent(t, got, "sensor_air_quality_aqi")
	assertAbsent(t, got, "sensor_air_quality_co2_ppm")
	assertAbsent(t, got, "sensor_air_quality_temperature_celsius")
}
