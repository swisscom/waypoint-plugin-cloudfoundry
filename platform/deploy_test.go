package platform

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"
	"testing"
)

func TestDeploy(t *testing.T) {
	p := Platform{}
	logger := hclog.New(nil)
	src := component.Source{
		App:  "app-name",
		Path: "path-name",
	}

	img := docker.Image{
		Image:    "some-image",
		Tag:      "latest",
		Location: nil,
	}

	deployConfig := component.DeploymentConfig{}
	ui := terminal.ConsoleUI(context.Background())

	p.Deploy(
		context.Background(),
		logger,
		&src,
		&img,
		&deployConfig,
		ui,
	)
}