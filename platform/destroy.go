package platform

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// DestroyFunc implements the Destroyer interface
func (p *Platform) DestroyFunc() interface{} {
	return p.destroy
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
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) destroy(ctx context.Context, ui terminal.UI, deployment *Deployment, source *component.Source) error {
	appName := source.App
	deploymentName := fmt.Sprintf("%v-%v", appName, deployment.Id)

	sg := ui.StepGroup()
	step := sg.Add("Connecting to Cloud Foundry")

	client, err := GetEnvClient()
	if err != nil {
		step.Abort()
		return fmt.Errorf("unable to create Cloud Foundry client: %v", err)
	}

	step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL))
	step.Done()

	orgGuid := deployment.OrganisationGUID
	spaceGuid := deployment.SpaceGUID

	step = sg.Add(fmt.Sprintf("Getting app info for %v", deploymentName))

	apps, _, err := client.GetApplications(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGuid},
	}, ccv3.Query{
		Key:    ccv3.SpaceGUIDFilter,
		Values: []string{spaceGuid},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{deploymentName},
	})
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to get app info: %v", err)
	}
	if len(apps) == 0 {
		step.Done()
		return nil
	}
	app := apps[0]
	step.Done()

	stepDescription := fmt.Sprintf("Deleting app routes for %v", deploymentName)
	step = sg.Add(stepDescription)
	routes, _, err := client.GetApplicationRoutes(app.GUID)
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to get app routes: %v", err)
	}
	for _, route := range routes {
		// only delete if it's the automatically created deployment route
		if route.Host == deploymentName {
			_, _, err = client.DeleteRoute(route.GUID)
			if err != nil {
				step.Update(fmt.Sprintf("%v [failed to delete route]", stepDescription))
			}
		}
	}
	step.Done()

	step = sg.Add(fmt.Sprintf("Deleting app %v", app.Name))
	_, _, err = client.DeleteApplication(app.GUID)
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to delete app: %v", err)
	}
	step.Done()

	return nil
}
