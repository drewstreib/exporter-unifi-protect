//nolint:testpackage // white-box: compares the collector's declared Descs against an upstream fixture
package internal

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// describedFamilies returns the set of fully-qualified metric names the
// collector declares via Describe.
func describedFamilies(t *testing.T) map[string]bool {
	t.Helper()

	fqNameRe := regexp.MustCompile(`fqName: "([^"]+)"`)

	ch := make(chan *prometheus.Desc, 256)
	NewCollector(nil, time.Minute, time.Second, true).Describe(ch)
	close(ch)

	out := make(map[string]bool)
	for desc := range ch {
		match := fqNameRe.FindStringSubmatch(desc.String())
		if match == nil {
			t.Fatalf("cannot parse fqName from %q", desc.String())
		}

		out[match[1]] = true
	}

	return out
}

// upstreamSensorFamilies parses the sensor_* metric family names from a captured
// Prometheus exposition fixture (the upstream exporter's /metrics output).
func upstreamSensorFamilies(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var names []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// "# HELP <name> <help...>"
		if len(fields) >= 3 && fields[0] == "#" && fields[1] == "HELP" && strings.HasPrefix(fields[2], "sensor_") {
			names = append(names, fields[2])
		}
	}

	if serr := scanner.Err(); serr != nil {
		t.Fatalf("scan fixture: %v", serr)
	}

	if len(names) == 0 {
		t.Fatal("no sensor_* families found in upstream fixture")
	}

	return names
}

// TestUpstreamMetricCoverage guards against silently dropping a metric that the
// original (merlindorin v0.0.8) exporter exported. Every sensor_* family it
// produced must still be declared by this collector.
func TestUpstreamMetricCoverage(t *testing.T) {
	ours := describedFamilies(t)

	for _, name := range upstreamSensorFamilies(t, "testdata/upstream-v0.0.8.metrics") {
		if !ours[name] {
			t.Errorf("regression: upstream metric %q is no longer exported", name)
		}
	}
}
