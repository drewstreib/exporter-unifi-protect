package internal

import (
	"context"

	"github.com/merlindorin/go-shared/pkg/net/do"
	"github.com/merlindorin/go-shared/pkg/net/rest"
)

// sensorsAPIPath is the UniFi Protect endpoint that lists every adopted sensor.
const sensorsAPIPath = "/proxy/protect/api/sensors"

// Sensor is an exporter-local model of a UniFi Protect sensor. It is decoded
// directly from the Protect API rather than relying on the go-unifi-protect
// Sensor type so that we can model fields that type does not expose yet (the
// air-quality block) and decode the many fields that are null on devices which
// do not support them (for example, environmental stats on the UP Air Quality
// sensor) as pointers, so absent readings can be skipped instead of exported as
// a misleading zero.
type Sensor struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	FirmwareVersion  string `json:"firmwareVersion"`
	HardwareRevision string `json:"hardwareRevision"`
	NvrMac           string `json:"nvrMac"`
	ModelKey         string `json:"modelKey"`
	MarketName       string `json:"marketName"`

	UpSince        int64 `json:"upSince"`
	LastSeen       int64 `json:"lastSeen"`
	ConnectedSince int64 `json:"connectedSince"`

	IsUpdating            bool `json:"isUpdating"`
	IsDownloadingFW       bool `json:"isDownloadingFW"`
	IsAdopting            bool `json:"isAdopting"`
	IsRestoring           bool `json:"isRestoring"`
	IsAdopted             bool `json:"isAdopted"`
	IsAdoptedByOther      bool `json:"isAdoptedByOther"`
	IsProvisioned         bool `json:"isProvisioned"`
	IsRebooting           bool `json:"isRebooting"`
	IsSSHEnabled          bool `json:"isSshEnabled"`
	CanAdopt              bool `json:"canAdopt"`
	IsAttemptingToConnect bool `json:"isAttemptingToConnect"`
	IsConnected           bool `json:"isConnected"`

	// Detection fields are null on devices that do not provide them.
	IsOpened               *bool  `json:"isOpened"`
	OpenStatusChangedAt    *int64 `json:"openStatusChangedAt"`
	MotionDetectedAt       *int64 `json:"motionDetectedAt"`
	LeakDetectedAt         *int64 `json:"leakDetectedAt"`
	ExternalLeakDetectedAt *int64 `json:"externalLeakDetectedAt"`

	Stats struct {
		Light       Measure `json:"light"`
		Humidity    Measure `json:"humidity"`
		Temperature Measure `json:"temperature"`
	} `json:"stats"`

	// AirQuality is only present on air-quality devices (UP Air Quality).
	AirQuality *AirQuality `json:"airQuality"`

	// BluetoothConnectionState is null on wired/air-quality devices.
	BluetoothConnectionState *struct {
		SignalQuality  float64 `json:"signalQuality"`
		SignalStrength float64 `json:"signalStrength"`
	} `json:"bluetoothConnectionState"`

	BatteryStatus struct {
		Percentage *float64 `json:"percentage"`
		IsLow      bool     `json:"isLow"`
	} `json:"batteryStatus"`
}

// Measure is a single UniFi Protect reading. Value is a pointer so that a null
// reading (status "unknown") can be distinguished from a real zero.
type Measure struct {
	Value  *float64 `json:"value"`
	Status string   `json:"status"`
}

// AirQuality holds the readings reported by the UP Air Quality sensor.
type AirQuality struct {
	AQI         Measure `json:"aqi"`
	Vape        Measure `json:"vape"`
	TVOC        Measure `json:"tvoc"`
	VOC         Measure `json:"voc"`
	CO2         Measure `json:"co2"`
	PM1p0       Measure `json:"pm1p0"`
	PM2p5       Measure `json:"pm2p5"`
	PM4p0       Measure `json:"pm4p0"`
	PM10p0      Measure `json:"pm10p0"`
	Humidity    Measure `json:"humidity"`
	Temperature Measure `json:"temperature"`
}

// listSensors fetches every sensor from the Protect API into sensors.
func listSensors(ctx context.Context, requester rest.Requester, sensors *[]Sensor) error {
	return requester.New(do.WithPath(sensorsAPIPath)).GET(ctx, do.WithUnmarshalBody(sensors))
}
