package release

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/cloudfoundry"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
)

// DestroyFunc implements the Destroyer interface
func (r *Releaser) DestroyFunc() interface{} {
	return r.destroy
}

// A DestroyFunc does not have a strict signature, you can define the parameters
// you need based on the Available parameters that the Waypoint SDK provides.
// Waypoint will automatically inject parameters as specified
// in the signature at run time.
//
// Available input parameters:
// - context.Context
// - *component.Source
// - *component.JobInfo
// - *component.DeploymentConfig
// - *datadir.Project
// - *datadir.App
// - *datadir.Component
// - hclog.Logger
// - terminal.UI
// - *component.LabelSet
//
// In addition to default input parameters the Deployment from the DeployFunc step
// can also be injected.
//
// The output parameters for PushFunc must be a Struct which can
// be serialized to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (r *Releaser) destroy(
	ctx context.Context,
	log hclog.Logger,
	ui terminal.UI,
	release *Release,
	source *component.Source,
	deployment *platform.Deployment,
) error {
	if r.config.StopOldInstances {
		// We want to stop old instnces of the application.
		sg := ui.StepGroup()
		step := sg.Add("Connecting to Cloud Foundry")

		client, err := cloudfoundry.New(log)
		if err != nil {
			step.Abort()
			return fmt.Errorf("unable to create Cloud Foundry client: %v", err)
		}

		step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL()))
		step.Done()

		orgGuid := deployment.OrganisationGUID
		spaceGuid := deployment.SpaceGUID

		step = sg.Add(fmt.Sprintf("Getting app info for %v", deployment.Name))

		apps, err := client.GetApplications(orgGuid, spaceGuid, deployment.Name)
		if err != nil {
			step.Abort()
			return fmt.Errorf("failed to get app info: %v", err)
		}
		if len(apps) == 0 {
			step.Abort()
			return fmt.Errorf("release failed, app not found")
		}
		step.Done()

	}
	return nil
}
