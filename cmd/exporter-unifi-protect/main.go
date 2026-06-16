package main

import (
	_ "embed"

	kongyaml "github.com/alecthomas/kong-yaml"

	"github.com/alecthomas/kong"
	"github.com/merlindorin/exporter-unifi-protect/cmd/exporter-unifi-protect/commads"
	"github.com/merlindorin/exporter-unifi-protect/internal/cli"
)

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
		kong.Configuration(kongyaml.Loader, "/etc/unifi-protect/config.yaml", "~/.hoomy/unifi-protect.yaml"),
	)

	ctx.FatalIfErrorf(ctx.Run(app.Commons))
}

type CMD struct {
	*cli.Commons
	Serve *commads.Serve `cmd:"serve"`
}
