package platform

import (
	"code.cloudfoundry.org/cli/resources"
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"os"
	"testing"
)

func TestPlatformGetOrgByName(t *testing.T) {
	p := Platform{
		config: Config{
			Organisation: os.Getenv("CF_ORG"),
			Space:        os.Getenv("CF_SPACE"),
		},
	}

	ui := terminal.ConsoleUI(context.Background())

	state := DeploymentState{}
	sg := ui.StepGroup()
	state.sg = &sg
	err := p.connectCloudFoundry(&state)
	if err != nil {
		t.Error(err)
		return
	}

	org, err := state.client.GetOrganization(p.config.Organisation)
	if err != nil {
		t.Error(err)
	}

	if org.Name != os.Getenv("CF_ORG") {
		t.Fatalf("invalid organization name returned, %s expected but %s returned",
			os.Getenv("CF_ORG"),
			org.Name,
		)
	}

	ui.Output("found org %s (%s)",
		org.Name,
		org.GUID,
		terminal.WithSuccessStyle(),
	)
}

func TestPlatformGetSpaceByName(t *testing.T) {
	p := Platform{
		config: Config{
			Organisation: os.Getenv("CF_ORG"),
			Space:        os.Getenv("CF_SPACE"),
		},
	}

	ui := terminal.ConsoleUI(context.Background())

	state := DeploymentState{}
	sg := ui.StepGroup()
	state.sg = &sg
	err := p.connectCloudFoundry(&state)
	if err != nil {
		t.Error(err)
		return
	}

	org := resources.Organization{
		GUID: os.Getenv("CF_ORG_GUID"),
		Name: p.config.Organisation,
	}

	space, err := state.client.GetSpaceByName(p.config.Space, org.GUID)
	if err != nil {
		t.Error(err)
	}

	if space.Name != os.Getenv("CF_SPACE") {
		t.Fatalf("invalid space name returned, %s expected but %s returned",
			os.Getenv("CF_SPACE"),
			space.Name,
		)
	}

	ui.Output("found space %s (%s)",
		space.Name,
		space.GUID,
		terminal.WithSuccessStyle(),
	)
}

func TestHealthCheck(t *testing.T) {
	p := Platform{
		config: Config{
			Organisation: os.Getenv("CF_ORG"),
			Space:        os.Getenv("CF_SPACE"),
		},
	}

	logger := hclog.New(hclog.DefaultOptions)
	p.log = logger

	ui := terminal.ConsoleUI(context.Background())

	deployment := Deployment{
		Url:              "https://unknown-url",
		Id:               "some-id",
		OrganisationGUID: os.Getenv("CF_ORG_GUID"),
		SpaceGUID:        os.Getenv("CF_SPACE_GUID"),
		AppGUID:          os.Getenv("CF_APP_GUID"),
		Name:             os.Getenv("CF_APP_NAME"),
	}

	status, err := p.Status(context.Background(),
		logger,
		&deployment,
		ui,
	)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("status=%v", status)
}
