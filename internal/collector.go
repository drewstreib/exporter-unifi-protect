//nolint:lll
package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	microsec = 1000
)

// sensorLister is the subset of the Protect client the collector needs.
type sensorLister interface {
	ListSensors(ctx context.Context) ([]Sensor, error)
}

type Collector struct {
	client  sensorLister
	timeout time.Duration

	// If true, any error encountered during collection is reported as an
	// invalid metric (see NewInvalidMetric). Otherwise, errors are ignored
	// and the collected metrics will be incomplete. (Possibly, no metrics
	// will be collected at all.) While that's usually not desired, it is
	// appropriate for the common "mix-in" of process metrics, where process
	// metrics are nice to have, but failing to collect them should not
	// disrupt the collection of the remaining metrics.
	reportErrors bool

	minDetectionSpan time.Duration

	sensorInfoGauge              *prometheus.Desc
	temperatureGauge             *prometheus.Desc
	lightGauge                   *prometheus.Desc
	humidityGauge                *prometheus.Desc
	batteryStatusPercentageGauge *prometheus.Desc
	bluetoothSignalStrengthGauge *prometheus.Desc
	bluetoothSignalQualityGauge  *prometheus.Desc
	isUpdatingGauge              *prometheus.Desc
	isDownloadingFWGauge         *prometheus.Desc
	isAdoptingGauge              *prometheus.Desc
	isRestoringGauge             *prometheus.Desc
	isAdoptedGauge               *prometheus.Desc
	isAdoptedByOtherGauge        *prometheus.Desc
	isProvisionedGauge           *prometheus.Desc
	isRebootingGauge             *prometheus.Desc
	isSSHEnabledGauge            *prometheus.Desc
	canAdoptGauge                *prometheus.Desc
	isAttemptingToConnectGauge   *prometheus.Desc
	isMotionDetectedGauge        *prometheus.Desc
	isOpenedGauge                *prometheus.Desc
	isConnectedGauge             *prometheus.Desc
	upSinceGauge                 *prometheus.Desc
	lastSeenGauge                *prometheus.Desc
	connectedSinceGauge          *prometheus.Desc
	leakDetectedAtGauge          *prometheus.Desc
	externalLeakDetectedAtGauge  *prometheus.Desc

	// Air-quality readings (UP Air Quality sensor).
	airQualityAQIGauge    *prometheus.Desc
	airQualityVapeGauge   *prometheus.Desc
	airQualityTVOCGauge   *prometheus.Desc
	airQualityVOCGauge    *prometheus.Desc
	airQualityCO2Gauge    *prometheus.Desc
	airQualityPM1p0Gauge  *prometheus.Desc
	airQualityPM2p5Gauge  *prometheus.Desc
	airQualityPM4p0Gauge  *prometheus.Desc
	airQualityPM10p0Gauge *prometheus.Desc
}

func NewCollector(client sensorLister, minDetectionSpan time.Duration, timeout time.Duration, reportError bool) *Collector {
	idName := []string{"id", "name"}

	return &Collector{
		client:       client,
		reportErrors: reportError,
		timeout:      timeout,

		minDetectionSpan: minDetectionSpan,

		sensorInfoGauge:              prometheus.NewDesc("sensor_info", "Sensor info.", []string{"id", "name", "firmwareVersion", "hardwareRevision", "nvr_mac", "brand", "type", "model", "market_name"}, nil),
		temperatureGauge:             prometheus.NewDesc("sensor_temperature_celsius", "Sensor monitor for temperature (input).", idName, nil),
		lightGauge:                   prometheus.NewDesc("sensor_light_lux", "Sensor monitor for light (input).", idName, nil),
		humidityGauge:                prometheus.NewDesc("sensor_humidity_percentage", "Sensor monitor for humidity (input).", idName, nil),
		batteryStatusPercentageGauge: prometheus.NewDesc("sensor_battery_status_percentage", "Sensor battery status.", []string{"id", "name", "is_low"}, nil),
		bluetoothSignalStrengthGauge: prometheus.NewDesc("sensor_bluetooth_signal_strength", "Sensor bluetooth signal strength (input).", idName, nil),
		bluetoothSignalQualityGauge:  prometheus.NewDesc("sensor_bluetooth_signal_quality", "Sensor bluetooth signal quality (input).", idName, nil),
		isUpdatingGauge:              prometheus.NewDesc("sensor_is_updating", "Sensor IsUpdatingGauge status (input).", idName, nil),
		isDownloadingFWGauge:         prometheus.NewDesc("sensor_is_downloading_f_w", "Sensor IsDownloadingFWGauge status (input).", idName, nil),
		isAdoptingGauge:              prometheus.NewDesc("sensor_is_adopting", "Sensor IsAdoptingGauge status (input).", idName, nil),
		isRestoringGauge:             prometheus.NewDesc("sensor_is_restoring", "Sensor IsRestoringGauge status (input).", idName, nil),
		isAdoptedGauge:               prometheus.NewDesc("sensor_is_adopted", "Sensor IsAdoptedGauge status (input).", idName, nil),
		isAdoptedByOtherGauge:        prometheus.NewDesc("sensor_is_adopted_by_other", "Sensor IsAdoptedByOtherGauge status (input).", idName, nil),
		isProvisionedGauge:           prometheus.NewDesc("sensor_is_provisioned", "Sensor IsProvisionedGauge status (input).", idName, nil),
		isRebootingGauge:             prometheus.NewDesc("sensor_is_rebooting", "Sensor IsRebootingGauge status (input).", idName, nil),
		isSSHEnabledGauge:            prometheus.NewDesc("sensor_is_ssh_enabled", "Sensor IsSshEnabledGauge status (input).", idName, nil),
		canAdoptGauge:                prometheus.NewDesc("sensor_can_adopt", "Sensor CanAdoptGauge status (input).", idName, nil),
		isAttemptingToConnectGauge:   prometheus.NewDesc("sensor_is_attempting_to_connect", "Sensor IsAttemptingToConnectGauge status (input).", idName, nil),
		isMotionDetectedGauge:        prometheus.NewDesc("sensor_is_motion_detected", "Sensor IsMotionDetectedGauge status (input).", []string{"id", "name", "detected_period"}, nil),
		isOpenedGauge:                prometheus.NewDesc("sensor_is_opened", "Sensor IsOpenedGauge status (input).", []string{"id", "name", "detected_period"}, nil),
		isConnectedGauge:             prometheus.NewDesc("sensor_is_connected", "Sensor IsConnectedGauge status (input).", idName, nil),
		upSinceGauge:                 prometheus.NewDesc("sensor_up_since_gauge", "Sensor UpSince status (input).", idName, nil),
		lastSeenGauge:                prometheus.NewDesc("sensor_last_seen_gauge", "Sensor LastSeen status (input).", idName, nil),
		connectedSinceGauge:          prometheus.NewDesc("sensor_connected_since_gauge", "Sensor ConnectedSince status (input).", idName, nil),
		leakDetectedAtGauge:          prometheus.NewDesc("sensor_leak_detected_at", "Sensor LeakDetectedAt status (input).", idName, nil),
		externalLeakDetectedAtGauge:  prometheus.NewDesc("sensor_external_leak_detected_at", "Sensor ExternalLeakDetectedAt status (input).", idName, nil),

		airQualityAQIGauge:    prometheus.NewDesc("sensor_air_quality_aqi", "Air quality index.", idName, nil),
		airQualityVapeGauge:   prometheus.NewDesc("sensor_air_quality_vape", "Air quality vape detection level.", idName, nil),
		airQualityTVOCGauge:   prometheus.NewDesc("sensor_air_quality_tvoc", "Air quality total volatile organic compounds (ppb).", idName, nil),
		airQualityVOCGauge:    prometheus.NewDesc("sensor_air_quality_voc", "Air quality volatile organic compounds index.", idName, nil),
		airQualityCO2Gauge:    prometheus.NewDesc("sensor_air_quality_co2_ppm", "Air quality CO2 concentration (ppm).", idName, nil),
		airQualityPM1p0Gauge:  prometheus.NewDesc("sensor_air_quality_pm1p0", "Air quality particulate matter PM1.0 (µg/m³).", idName, nil),
		airQualityPM2p5Gauge:  prometheus.NewDesc("sensor_air_quality_pm2p5", "Air quality particulate matter PM2.5 (µg/m³).", idName, nil),
		airQualityPM4p0Gauge:  prometheus.NewDesc("sensor_air_quality_pm4p0", "Air quality particulate matter PM4.0 (µg/m³).", idName, nil),
		airQualityPM10p0Gauge: prometheus.NewDesc("sensor_air_quality_pm10p0", "Air quality particulate matter PM10 (µg/m³).", idName, nil),
	}
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	descs <- c.sensorInfoGauge
	descs <- c.temperatureGauge
	descs <- c.lightGauge
	descs <- c.humidityGauge
	descs <- c.batteryStatusPercentageGauge
	descs <- c.bluetoothSignalStrengthGauge
	descs <- c.bluetoothSignalQualityGauge
	descs <- c.isUpdatingGauge
	descs <- c.isDownloadingFWGauge
	descs <- c.isAdoptingGauge
	descs <- c.isRestoringGauge
	descs <- c.isAdoptedGauge
	descs <- c.isAdoptedByOtherGauge
	descs <- c.isProvisionedGauge
	descs <- c.isRebootingGauge
	descs <- c.isSSHEnabledGauge
	descs <- c.canAdoptGauge
	descs <- c.isAttemptingToConnectGauge
	descs <- c.isMotionDetectedGauge
	descs <- c.isOpenedGauge
	descs <- c.isConnectedGauge
	descs <- c.upSinceGauge
	descs <- c.lastSeenGauge
	descs <- c.connectedSinceGauge
	descs <- c.leakDetectedAtGauge
	descs <- c.externalLeakDetectedAtGauge
	descs <- c.airQualityAQIGauge
	descs <- c.airQualityVapeGauge
	descs <- c.airQualityTVOCGauge
	descs <- c.airQualityVOCGauge
	descs <- c.airQualityCO2Gauge
	descs <- c.airQualityPM1p0Gauge
	descs <- c.airQualityPM2p5Gauge
	descs <- c.airQualityPM4p0Gauge
	descs <- c.airQualityPM10p0Gauge
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	sensors, err := c.client.ListSensors(ctx)
	if err != nil {
		c.reportError(ch, nil, err)
		return
	}

	for i := range sensors {
		c.collectSensor(ch, &sensors[i])
	}
}

func (c *Collector) collectSensor(ch chan<- prometheus.Metric, sensor *Sensor) {
	ch <- prometheus.MustNewConstMetric(c.sensorInfoGauge, prometheus.GaugeValue, 1, sensor.ID, sensor.Name, sensor.FirmwareVersion, sensor.HardwareRevision, sensor.NvrMac, "unifi", sensor.Type, sensor.ModelKey, sensor.MarketName)

	// Environmental readings are null on devices that do not provide them. The
	// UP Air Quality sensor reports temperature/humidity under airQuality
	// instead of stats, so fall back to that block to keep them on the common
	// sensor_temperature_celsius / sensor_humidity_percentage metrics rather
	// than a device-specific name. Only export readings that are present.
	temperature, humidity := sensor.Stats.Temperature, sensor.Stats.Humidity
	if aq := sensor.AirQuality; aq != nil {
		if temperature.Value == nil {
			temperature = aq.Temperature
		}

		if humidity.Value == nil {
			humidity = aq.Humidity
		}
	}

	c.measure(ch, c.temperatureGauge, temperature, sensor.ID, sensor.Name)
	c.measure(ch, c.humidityGauge, humidity, sensor.ID, sensor.Name)
	c.measure(ch, c.lightGauge, sensor.Stats.Light, sensor.ID, sensor.Name)

	if bt := sensor.BluetoothConnectionState; bt != nil {
		ch <- prometheus.MustNewConstMetric(c.bluetoothSignalQualityGauge, prometheus.GaugeValue, bt.SignalQuality, sensor.ID, sensor.Name)
		ch <- prometheus.MustNewConstMetric(c.bluetoothSignalStrengthGauge, prometheus.GaugeValue, bt.SignalStrength, sensor.ID, sensor.Name)
	}

	if p := sensor.BatteryStatus.Percentage; p != nil {
		ch <- prometheus.MustNewConstMetric(c.batteryStatusPercentageGauge, prometheus.GaugeValue, *p, sensor.ID, sensor.Name, strconv.FormatBool(sensor.BatteryStatus.IsLow))
	}

	ch <- prometheus.MustNewConstMetric(c.isUpdatingGauge, prometheus.GaugeValue, boolToInt(sensor.IsUpdating), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isDownloadingFWGauge, prometheus.GaugeValue, boolToInt(sensor.IsDownloadingFW), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isAdoptingGauge, prometheus.GaugeValue, boolToInt(sensor.IsAdopting), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isRestoringGauge, prometheus.GaugeValue, boolToInt(sensor.IsRestoring), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isAdoptedGauge, prometheus.GaugeValue, boolToInt(sensor.IsAdopted), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isAdoptedByOtherGauge, prometheus.GaugeValue, boolToInt(sensor.IsAdoptedByOther), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isProvisionedGauge, prometheus.GaugeValue, boolToInt(sensor.IsProvisioned), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isRebootingGauge, prometheus.GaugeValue, boolToInt(sensor.IsRebooting), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isSSHEnabledGauge, prometheus.GaugeValue, boolToInt(sensor.IsSSHEnabled), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.canAdoptGauge, prometheus.GaugeValue, boolToInt(sensor.CanAdopt), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isAttemptingToConnectGauge, prometheus.GaugeValue, boolToInt(sensor.IsAttemptingToConnect), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.isConnectedGauge, prometheus.GaugeValue, boolToInt(sensor.IsConnected), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.upSinceGauge, prometheus.GaugeValue, float64(sensor.UpSince), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.lastSeenGauge, prometheus.GaugeValue, float64(sensor.LastSeen), sensor.ID, sensor.Name)
	ch <- prometheus.MustNewConstMetric(c.connectedSinceGauge, prometheus.GaugeValue, float64(sensor.ConnectedSince), sensor.ID, sensor.Name)

	period := fmt.Sprintf("%.0f", c.minDetectionSpan.Seconds())

	if sensor.MotionDetectedAt != nil {
		detected := time.Now().Before(time.UnixMicro(*sensor.MotionDetectedAt * microsec).Add(c.minDetectionSpan))
		ch <- prometheus.MustNewConstMetric(c.isMotionDetectedGauge, prometheus.GaugeValue, boolToInt(detected), sensor.ID, sensor.Name, period)
	}

	if sensor.IsOpened != nil && sensor.OpenStatusChangedAt != nil {
		opened := time.Now().Before(time.UnixMicro(*sensor.OpenStatusChangedAt*microsec).Add(c.minDetectionSpan)) || *sensor.IsOpened
		ch <- prometheus.MustNewConstMetric(c.isOpenedGauge, prometheus.GaugeValue, boolToInt(opened), sensor.ID, sensor.Name, period)
	}

	if sensor.LeakDetectedAt != nil {
		ch <- prometheus.MustNewConstMetric(c.leakDetectedAtGauge, prometheus.GaugeValue, float64(*sensor.LeakDetectedAt), sensor.ID, sensor.Name)
	}

	if sensor.ExternalLeakDetectedAt != nil {
		ch <- prometheus.MustNewConstMetric(c.externalLeakDetectedAtGauge, prometheus.GaugeValue, float64(*sensor.ExternalLeakDetectedAt), sensor.ID, sensor.Name)
	}

	c.collectAirQuality(ch, sensor)
}

func (c *Collector) collectAirQuality(ch chan<- prometheus.Metric, sensor *Sensor) {
	aq := sensor.AirQuality
	if aq == nil {
		return
	}

	c.measure(ch, c.airQualityAQIGauge, aq.AQI, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityVapeGauge, aq.Vape, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityTVOCGauge, aq.TVOC, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityVOCGauge, aq.VOC, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityCO2Gauge, aq.CO2, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityPM1p0Gauge, aq.PM1p0, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityPM2p5Gauge, aq.PM2p5, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityPM4p0Gauge, aq.PM4p0, sensor.ID, sensor.Name)
	c.measure(ch, c.airQualityPM10p0Gauge, aq.PM10p0, sensor.ID, sensor.Name)
}

// measure exports a single reading, skipping it when the value is absent (null)
// so that unsupported readings are not reported as a misleading zero.
func (c *Collector) measure(ch chan<- prometheus.Metric, desc *prometheus.Desc, m Measure, labels ...string) {
	if m.Value == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, *m.Value, labels...)
}

func (c *Collector) reportError(ch chan<- prometheus.Metric, desc *prometheus.Desc, err error) {
	if !c.reportErrors {
		return
	}

	if desc == nil {
		desc = prometheus.NewInvalidDesc(err)
	}

	ch <- prometheus.NewInvalidMetric(desc, err)
}

func boolToInt(updating bool) float64 {
	if updating {
		return 1
	}

	return 0
}
