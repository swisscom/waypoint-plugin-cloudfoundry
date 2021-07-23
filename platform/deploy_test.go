package platform

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"
	"os"
	"testing"
)

func TestDeploy(t *testing.T) {
	p := Platform{
		config: Config{
			Organisation: os.Getenv("CF_ORG"),
			Space:        os.Getenv("CF_SPACE"),
			Domain:       os.Getenv("CF_DOMAIN"),
		},
	}
	logger := hclog.New(nil)
	src := component.Source{
		App:  "app-name",
		Path: "path-name",
	}

	img := docker.Image{
		Image:    os.Getenv("CF_IMAGE"),
		Tag:      "latest",
		Location: nil,
	}

	deployConfig := component.DeploymentConfig{}
	ui := terminal.ConsoleUI(context.Background())

	deployment, err := p.Deploy(
		context.Background(),
		logger,
		&src,
		&img,
		&deployConfig,
		ui,
	)

	if err != nil {
		t.Fatal(err)
		return
	}

	if deployment.Name != "app-name" {
		t.Fatalf("expected deployment.Name to be app-name, but got %s instead", deployment.Name)
	}
}
