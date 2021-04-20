package main

import (
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/release"
)

func main() {
	// sdk.Main allows you to register the components which should
	// be included in your plugin
	// Main sets up all the go-plugin requirements

	sdk.Main(sdk.WithComponents(
		// Comment out any components which are not
		// required for your plugin
		// &builder.Builder{},
		// &registry.Registry{},
		&platform.Platform{},
		&release.ReleaseManager{},
	))
}
