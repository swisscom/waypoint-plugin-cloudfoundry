package release

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
)

type Config struct {
	Domain   string `hcl:"domain"`
	Hostname string `hcl:"hostname,optional"`
}

type Manager struct {
	config Config
}

// Config Implement Configurable
func (rm *Manager) Config() (interface{}, error) {
	return &rm.config, nil
}

// ReleaseFunc Implement Builder
func (rm *Manager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.release
}

//
// The output parameters for ReleaseFunc must be a Struct which can
// be serialized to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *Manager) release(
	ctx context.Context,
	log hclog.Logger,
	ui terminal.UI,
	src *component.Source,
	deployment *platform.Deployment,
) (*Release, error) {
	var release Release

	var hostname string
	if rm.config.Hostname != "" {
		hostname = rm.config.Hostname
	} else {
		hostname = src.App
	}

	sg := ui.StepGroup()
	step := sg.Add("Connecting to Cloud Foundry")

	client, err := platform.GetEnvClient()
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("unable to create Cloud Foundry client: %v", err)
	}

	step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL))
	step.Done()

	orgGuid := deployment.OrganisationGUID
	spaceGuid := deployment.SpaceGUID

	step = sg.Add(fmt.Sprintf("Getting app info for %v", deployment.Name))

	apps, _, err := client.GetApplications(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGuid},
	}, ccv3.Query{
		Key:    ccv3.SpaceGUIDFilter,
		Values: []string{spaceGuid},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{deployment.Name},
	})
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to get app info: %v", err)
	}
	if len(apps) == 0 {
		step.Abort()
		return nil, fmt.Errorf("release failed, app not found")
	}
	step.Done()

	if rm.config.Hostname == "" {
		rm.config.Hostname = src.App
	}

	routeUrl := fmt.Sprintf("%v.%v", rm.config.Hostname, rm.config.Domain)
	step = sg.Add(fmt.Sprintf("Binding route %v to deployment", routeUrl))
	domains, _, err := client.GetDomains(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{rm.config.Domain},
	})
	if err != nil || len(domains) == 0 {
		step.Abort()
		return nil, fmt.Errorf("failed to get specified domain: %v", err)
	}
	domain := domains[0]

	// Check if route exists already
	routes, _, err := client.GetRoutes(ccv3.Query{
		Key:    ccv3.DomainGUIDFilter,
		Values: []string{domain.GUID},
	}, ccv3.Query{
		Key:    ccv3.HostsFilter,
		Values: []string{hostname},
	})
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed checking if route exists already: %v", err)
	}
	var route resources.Route
	if len(routes) == 0 {
		route, _, err = client.CreateRoute(resources.Route{
			DomainGUID: domain.GUID,
			SpaceGUID:  spaceGuid,
			Host:       hostname,
			Destinations: []resources.RouteDestination{{
				App: resources.RouteDestinationApp{
					GUID: deployment.AppGUID,
				},
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed creating route: %v", err)
		}
	} else {
		route = routes[0]
	}

	// Also map
	_, err = client.MapRoute(route.GUID, deployment.AppGUID)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to map route: %v", err)
	}
	step.Done()
	release.Url = fmt.Sprintf("%v://%v", route.Protocol, route.URL)

	// Unmap all other applications
	for _, destination := range route.Destinations {
		_, err = client.UnmapRoute(route.GUID, destination.GUID)
		if err != nil {
			return nil, fmt.Errorf("failed to unmap route from destination app with GUID %v", destination.App.GUID)
		}
	}

	return &release, nil
}

func (r *Release) URL() string { return r.Url }

var _ component.Release = (*Release)(nil)
var _ component.ReleaseManager = (*Manager)(nil)
