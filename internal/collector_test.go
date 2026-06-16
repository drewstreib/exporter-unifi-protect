//nolint:testpackage // white-box test: exercises unexported collectSensor without a network round-trip
package internal

import (
	"encoding/json"
	"math"
	"os"
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

// loadSensor decodes a real captured Protect payload from testdata. These
// fixtures double as reference samples of the API response for each device type.
func loadSensor(t *testing.T, path string) *Sensor {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}

	return mustDecode(t, string(raw))
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

func TestCollectAirQualitySensor(t *testing.T) {
	c := NewCollector(nil, time.Minute, time.Second, true)
	got := collect(t, c, loadSensor(t, "testdata/up-airquality.json"))

	// Air-quality-only readings are exported under sensor_air_quality_*.
	assertValue(t, got, "sensor_air_quality_aqi", 7)
	assertValue(t, got, "sensor_air_quality_tvoc", 5.8)
	assertValue(t, got, "sensor_air_quality_voc", 67)
	assertValue(t, got, "sensor_air_quality_co2_ppm", 452)
	assertValue(t, got, "sensor_air_quality_pm1p0", 0.8)
	assertValue(t, got, "sensor_air_quality_pm2p5", 1.79)
	assertValue(t, got, "sensor_air_quality_pm4p0", 2.5)
	assertValue(t, got, "sensor_air_quality_pm10p0", 2.9)

	// Temperature/humidity from the airQuality block are surfaced on the common
	// metrics (same names the USL sensors use), not air-quality-specific ones.
	assertValue(t, got, "sensor_temperature_celsius", 25.4)
	assertValue(t, got, "sensor_humidity_percentage", 51)
	assertAbsent(t, got, "sensor_air_quality_temperature_celsius")
	assertAbsent(t, got, "sensor_air_quality_humidity_percentage")

	// Generic device metrics still come through.
	assertValue(t, got, "sensor_info", 1)
	assertValue(t, got, "sensor_is_connected", 1)

	// Readings the device genuinely does not provide must not be exported as a
	// misleading zero.
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
