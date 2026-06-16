// Package cli provides the small command-line and web plumbing this exporter
// needs (version/licence subcommands, logger setup, a build-info collector, an
// external-URL helper, and a zap adapter for the exporter-toolkit web server).
// It replaces the equivalent helpers previously imported from go-shared.
package cli

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Commons holds the flags and subcommands shared by every command.
type Commons struct {
	Development bool   `short:"D" env:"DEBUG,DEV,DEVELOPMENT" help:"Set to true to enable development mode with debug-level logging."`
	Level       string `short:"l" env:"LOG_LEVEL" help:"Specify the logging level, options are: debug, info, warn, error, fatal." default:"info"`

	Version Version `cmd:"" help:"Display version information."`
	Licence Licence `cmd:"" help:"Show the application's licence."`
}

// Logger builds a zap.Logger from the Development and Level flags.
func (c *Commons) Logger() (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(c.Level)
	if err != nil {
		return nil, fmt.Errorf("cannot parse logger level %q: %w", c.Level, err)
	}

	config := zap.NewProductionConfig()
	if c.Development {
		config = zap.NewDevelopmentConfig()
		level = zapcore.DebugLevel
	}

	config.Level = zap.NewAtomicLevelAt(level)

	return config.Build()
}

// BuildInfo holds the build details injected via ldflags.
type BuildInfo struct {
	name        string
	version     string
	commit      string
	buildSource string
	date        string
}

// NewBuildInfo creates a BuildInfo from the given details.
func NewBuildInfo(name, version, commit, buildSource, date string) BuildInfo {
	return BuildInfo{name: name, version: version, commit: commit, buildSource: buildSource, date: date}
}

func (b BuildInfo) Name() string        { return b.name }
func (b BuildInfo) Version() string     { return b.version }
func (b BuildInfo) Commit() string      { return b.commit }
func (b BuildInfo) BuildSource() string { return b.buildSource }
func (b BuildInfo) Date() string        { return b.date }

func (b BuildInfo) String() string {
	return fmt.Sprintf(
		"name=%s, version=%s, commit=%s, buildDate=%s, buildSource=%s",
		b.name, b.version, b.commit, b.date, b.buildSource,
	)
}

// Version is the "version" subcommand. BuildInfo's fields are unexported so kong
// does not turn them into flags of the command.
type Version struct {
	BuildInfo
}

// NewVersion builds the version subcommand from the given build details.
func NewVersion(name, version, commit, buildSource, date string) Version {
	return Version{NewBuildInfo(name, version, commit, buildSource, date)}
}

// Run prints the build information.
func (v Version) Run() error {
	//nolint:forbidigo // printing version to stdout is intended
	fmt.Print(v.BuildInfo.String())
	return nil
}

// Licence is the "licence" subcommand.
type Licence struct {
	content string
}

// NewLicence creates a Licence with the given content.
func NewLicence(s string) Licence {
	return Licence{content: s}
}

// Run prints the licence content.
func (l Licence) Run() error {
	//nolint:forbidigo // printing licence to stdout is intended
	fmt.Printf("%s", l.content)
	return nil
}

// NewBuildInfoCollector returns a prometheus collector exposing a single
// build_info gauge labelled with the build details.
func NewBuildInfoCollector(b BuildInfo) prometheus.Collector {
	return &buildInfoCollector{
		desc: prometheus.NewDesc(
			"build_info",
			"Information about the binary build.",
			nil,
			prometheus.Labels{
				"name":        b.Name(),
				"version":     b.Version(),
				"date":        b.Date(),
				"buildSource": b.BuildSource(),
				"commit":      b.Commit(),
			},
		),
	}
}

type buildInfoCollector struct {
	desc *prometheus.Desc
}

func (c *buildInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *buildInfoCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1)
}

// HTTPLogger adapts a *zap.Logger to the go-kit log.Logger interface
// (Log(keyvals ...interface{}) error) that exporter-toolkit's web package
// expects, without taking a direct dependency on go-kit/log.
type HTTPLogger struct {
	service string
	sugar   *zap.SugaredLogger
}

// NewHTTPLogger adapts logger for the web server, tagging entries with service.
func NewHTTPLogger(service string, logger *zap.Logger) HTTPLogger {
	return HTTPLogger{service: service, sugar: logger.Sugar()}
}

// Log implements the go-kit log.Logger interface.
func (h HTTPLogger) Log(keyvals ...interface{}) error {
	h.sugar.Infow(h.service, keyvals...)
	return nil
}

func startsOrEndsWithQuote(s string) bool {
	return strings.HasPrefix(s, "\"") || strings.HasPrefix(s, "'") ||
		strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'")
}

// ComputeExternalURL computes a sanitized external URL, inferring unset parts
// from the hostname and the given listen address.
func ComputeExternalURL(u, listenAddr string) (*url.URL, error) {
	if u == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}

		_, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return nil, err
		}

		u = net.JoinHostPort(hostname, port)
	}

	if startsOrEndsWithQuote(u) {
		return nil, errors.New("URL must not begin or end with quotes")
	}

	eu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	ppref := strings.TrimRight(eu.Path, "/")
	if ppref != "" && !strings.HasPrefix(ppref, "/") {
		ppref = "/" + ppref
	}

	eu.Path = ppref

	return eu, nil
}
