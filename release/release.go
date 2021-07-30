package release

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	proto "github.com/hashicorp/waypoint-plugin-sdk/proto/gen"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/cloudfoundry"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
)

type Config struct {
	Domain           string   `hcl:"domain"`
	Hostname         string   `hcl:"hostname,optional"`
	AdditionalRoutes []string `hcl:"additional_routes,optional"`
}

type Releaser struct {
	config Config
	log hclog.Logger
}

// Config Implement Configurable
func (r *Releaser) Config() (interface{}, error) {
	return &r.config, nil
}

// ReleaseFunc Implement Builder
func (r *Releaser) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return r.Release
}

// StatusFunc Implements the Status check for the Release
func (r *Releaser) StatusFunc() interface{} {
	return r.Status
}

// Release The output parameters for ReleaseFunc must be a Struct which can be serialized
// to Protocol Buffers binary format and an error. This Output Value will be made
// available for other functions as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.

func (r *Releaser) Release(
	ctx context.Context,
	log hclog.Logger,
	ui terminal.UI,
	src *component.Source,
	deployment *platform.Deployment,
) (*Release, error) {
	var release Release
	var hostname string

	if r.config.Hostname != "" {
		hostname = r.config.Hostname
	}

	sg := ui.StepGroup()
	step := sg.Add("Connecting to Cloud Foundry")

	client, err := cloudfoundry.New(log)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("unable to create Cloud Foundry client: %v", err)
	}

	step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL()))
	step.Done()

	orgGuid := deployment.OrganisationGUID
	spaceGuid := deployment.SpaceGUID

	step = sg.Add(fmt.Sprintf("Getting app info for %v", deployment.Name))

	apps, err := client.GetApplications(orgGuid, spaceGuid, deployment.Name)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to get app info: %v", err)
	}
	if len(apps) == 0 {
		step.Abort()
		return nil, fmt.Errorf("release failed, app not found")
	}
	step.Done()

	if r.config.Hostname == "" {
		r.config.Hostname = src.App
	}

	routeUrl := fmt.Sprintf("%v.%v", r.config.Hostname, r.config.Domain)
	step = sg.Add(fmt.Sprintf("Binding route %v to deployment", routeUrl))
	domains, err := client.GetDomainsByName(r.config.Domain)
	if err != nil || len(domains) == 0 {
		step.Abort()
		return nil, fmt.Errorf("failed to get specified domain: %v", err)
	}
	domain := domains[0]


	// Map original route, if not empty
	if hostname != "" {
		route, err := client.UpsertRoute(hostname, domain, deployment.SpaceGUID)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("failed to get or create route: %v", err)
		}

		// Map route
		err = client.MapRoute(route.GUID, deployment.AppGUID)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("failed to map route: %v", err)
		}
		step.Done()
		release.Url = fmt.Sprintf("%v://%v", route.Protocol, route.URL)

		step = sg.Add("unmapping other applications")
		// Unmap all others applications
		for _, destination := range route.Destinations {
			step.Update(fmt.Sprintf("unmapping %v", destination.App.GUID))
			err = client.UnmapRoute(route.GUID, destination.GUID)
			if err != nil {
				return nil, fmt.Errorf("failed to unmap route from destination app with GUID %v", destination.App.GUID)
			}
		}
		step.Done()
	}


	step = sg.Add("mapping additional routes (if available)")
	for _, additionalRoute := range r.config.AdditionalRoutes {
		route, err := client.UpsertRoute(additionalRoute, domain, deployment.SpaceGUID)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("failed to get or create route: %v", err)
		}

		step.Update(fmt.Sprintf("mapping %v", route))
		err = client.MapRoute(route.GUID, deployment.AppGUID)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to map route %v to app %v: %v",
				route,
				deployment.AppGUID,
				err,
			)
		}

		step = sg.Add(fmt.Sprintf("unmapping previous app from %v", route))

		// Unmap other routes associated with this one
		for _, dest := range route.Destinations {
			step.Update("unmapping %v", dest.App.GUID)
			err = client.UnmapRoute(route.GUID, dest.GUID)
			if err != nil {
				return nil, fmt.Errorf(
					"unable to unmap route %v from app %v: %v",
					route,
					dest.App.GUID,
					err,
				)
			}
		}
		step.Done()
	}
	step.Done()



	return &release, nil
}

func (r *Releaser) listWarnings(warn ccv3.Warnings) {
	if len(warn) > 0 {
		for _, w := range warn {
			r.log.Warn("Cloud Foundry warning", "warning", w)
		}
	}
}

func (r *Releaser) Status(
	ctx context.Context,
	log hclog.Logger,
	release *Release,
	ui terminal.UI,
) (*proto.StatusReport, error) {
	if release.RouteGuid == "" {
		return nil, fmt.Errorf("route GUID cannot be empty")
	}
	r.log = log

	sg := ui.StepGroup()
	step := sg.Add("getting health for release")
	defer step.Abort()

	client, err := cloudfoundry.New(log)
	if err != nil {
		return nil, fmt.Errorf("unable to get Cloud Foundry client: %v", err)
	}

	routeGuid := release.RouteGuid

	route, err := client.GetRoute(routeGuid)
	if err != nil {
		return nil,
		fmt.Errorf(
			"unable to get routes for route GUID %s: %v",
			routeGuid,
			err,
		)
	}
	destinations := route.Destinations

	if len(destinations) == 0 {
		// No destinations = 404 !

	}

	/*(for _, dest := range destinations {
		dest.
	}*/

	step.Done()
	return nil, nil
}

func (r *Release) URL() string { return r.Url }

var _ component.Release = (*Release)(nil)
var _ component.ReleaseManager = (*Releaser)(nil)
var _ component.Status = (*Releaser)(nil)