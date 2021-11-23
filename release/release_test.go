package release

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
	"os"
	"strings"
	"testing"
)

func TestRelease(t *testing.T) {
	p := platform.Platform{}
	c, err := p.Config()
	if err != nil {
		t.Fatal(err)
	}
	*c.(*platform.Config) = platform.Config{
		Organisation:             os.Getenv("CF_ORG"),
		Space:                    os.Getenv("CF_SPACE"),
		Domain:                   os.Getenv("CF_DOMAIN"),
		DeploymentTimeoutSeconds: "10s",
	}

	logger := hclog.New(nil)
	src := component.Source{
		App:  "app-name",
		Path: "path-name",
	}

	ui := terminal.ConsoleUI(context.Background())

	r := Releaser{}
	r.config = Config{
		Domain:           os.Getenv("CF_DOMAIN"),
		Hostname:         os.Getenv("CF_HOSTNAME"),
		AdditionalRoutes: strings.Split(os.Getenv("CF_ADDITIONAL_ROUTES"), ","),
	}

	deployment := platform.Deployment{
		Url:              "https://some-url",
		Id:               "some-id",
		OrganisationGUID: os.Getenv("CF_ORG_GUID"),
		SpaceGUID:        os.Getenv("CF_SPACE_GUID"),
		AppGUID:          os.Getenv("CF_APP_GUID"),
		Name:             os.Getenv("CF_APP_NAME"),
	}

	release, err := r.Release(
		context.Background(),
		logger,
		ui,
		&src,
		&deployment,
	)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("ok: %v", release)
}

func TestStatus(t *testing.T) {
	p := platform.Platform{}
	c, err := p.Config()
	if err != nil {
		t.Fatal(err)
	}
	*c.(*platform.Config) = platform.Config{
		Organisation:             os.Getenv("CF_ORG"),
		Space:                    os.Getenv("CF_SPACE"),
		Domain:                   os.Getenv("CF_DOMAIN"),
		DeploymentTimeoutSeconds: "10s",
	}

	logger := hclog.New(nil)
	release := Release{
		Url: fmt.Sprintf(
			"https://%s.%s",
			os.Getenv("CF_HOSTNAME"),
			os.Getenv("CF_DOMAIN"),
		),
		RouteGuid: os.Getenv("CF_ROUTE_GUID"),
	}

	r := Releaser{
		config: Config{
			Domain:           os.Getenv("CF_DOMAIN"),
			Hostname:         os.Getenv("CF_HOSTNAME"),
			AdditionalRoutes: strings.Split(os.Getenv("CF_ADDITIONAL_ROUTES"), ","),
		},
		log: logger,
	}

	ui := terminal.ConsoleUI(context.Background())

	statusReport, err := r.Status(
		context.Background(),
		logger,
		&release,
		ui,
	)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("statusReport=+%v", statusReport)
}
