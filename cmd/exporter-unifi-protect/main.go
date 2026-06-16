package main

import (
	_ "embed"
	"fmt"
	"os"

	kongyaml "github.com/alecthomas/kong-yaml"

	"github.com/alecthomas/kong"
	"github.com/drewstreib/exporter-unifi-protect/cmd/exporter-unifi-protect/commads"
	"github.com/drewstreib/exporter-unifi-protect/internal/cli"
)

// envPrefix is prepended to every flag name to derive its environment variable
// (for example --web.listen-addresses -> UNIFI_WEB_LISTEN_ADDRESSES). Flags
// with an explicit env tag (UNIFI_USERNAME, UNIFI_PASSWORD, UNIFI_HOST, ...)
// keep their own name.
const envPrefix = "UNIFI"

// dotEnvFile is loaded from the working directory at startup, if present.
const dotEnvFile = ".env"

const (
	name        = "unifi-protect"
	description = "Exporter for Unifi protect"
)

//nolint:gochecknoglobals // these global variables exist to be overridden during build
var (
	license string

	version     = "dev"
	commit      = "dirty"
	date        = "latest"
	buildSource = "source"
)

func main() {
	if err := cli.LoadDotEnv(dotEnvFile); err != nil {
		fmt.Fprintf(os.Stderr, "cannot load %s: %v\n", dotEnvFile, err)
		os.Exit(1)
	}

	app := CMD{
		Commons: &cli.Commons{
			Version: cli.NewVersion(name, version, commit, buildSource, date),
			Licence: cli.NewLicence(license),
		},
		Serve: &commads.Serve{},
	}

	ctx := kong.Parse(
		&app,
		kong.Name(name),
		kong.Description(description),
		kong.UsageOnError(),
		kong.DefaultEnvars(envPrefix),
		kong.Configuration(kongyaml.Loader, "/etc/unifi-protect/config.yaml", "~/.hoomy/unifi-protect.yaml"),
	)

	ctx.FatalIfErrorf(ctx.Run(app.Commons))
}

type CMD struct {
	*cli.Commons
	Serve *commads.Serve `cmd:"serve"`
}
