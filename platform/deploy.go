package platform

import (
	"code.cloudfoundry.org/cli/types"
	"context"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	"time"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/resources"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"
)

type PlatformConfig struct {
	ApiUrl      string `hcl:"api_url"`
	EncodedAuth string `hcl:"encoded_auth"`
}

type QuotaConfig struct {
	Memory    string `hcl:"memory,optional"`
	Disk      string `hcl:"disk,optional"`
	Instances uint64 `hcl:"instances,optional"`
}

type Config struct {
	Organisation      string            `hcl:"organisation"`
	Space             string            `hcl:"space"`
	DockerEncodedAuth string            `hcl:"docker_encoded_auth,optional"`
	Domain            string            `hcl:"domain"`
	Quota             *QuotaConfig      `hcl:"quota,block"`
	Env               map[string]string `hcl:"env,optional"`
	EnvFromFile       string            `hcl:"envFromFile,optional"`
}

type Platform struct {
	config Config
}

type UserPasswordCredentials struct {
	Username string
	Password string
}

// Implement Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// Implement ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	_, ok := config.(*Config)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *Config as parameter")
	}

	return nil
}

// Implement Builder
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
}

// A BuildFunc does not have a strict signature, you can define the parameters
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

// In addition to default input parameters the registry.Artifact from the Build step
// can also be injected.
//
// The output parameters for BuildFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (b *Platform) deploy(ctx context.Context, log hclog.Logger, ui terminal.UI, img *docker.Image, job *component.JobInfo, source *component.Source, deploymentConfig *component.DeploymentConfig) (*Deployment, error) {
	// Create result
	var deployment Deployment
	id, err := component.Id()
	if err != nil {
		return nil, err
	}
	deployment.Id = id

	sg := ui.StepGroup()

	// Parse quantities
	step := sg.Add("Validating parameters")
	var diskMB, memoryMB, instances uint64

	if b.config.Quota != nil {
		log.Debug("quota is not nil")
		if b.config.Quota.Instances > 0 {
			instances = b.config.Quota.Instances
		}
		if b.config.Quota.Memory != "" {
			log.Debug("quota memory", "quota_memory", b.config.Quota.Memory)
			memoryMB, err = parseQuantity(b.config.Quota.Memory)
			if err != nil {
				step.Abort()
				return nil, fmt.Errorf("unable to parse memory: %v", err)
			}
			log.Debug("quota parsed", "quota_parsed", memoryMB)
		}
		if b.config.Quota.Disk != "" {
			log.Debug("quota disk", "quota_disk", b.config.Quota.Disk)
			diskMB, err = parseQuantity(b.config.Quota.Disk)
			if err != nil {
				return nil, fmt.Errorf("unable to parse disk: %v", err)
			}
			log.Debug("quota parsed disk", "quota_parsed", diskMB)
		}
	}
	step.Done()

	appName := source.App
	deployment.Name = fmt.Sprintf("%v-%v", appName, deployment.Id)

	step = sg.Add("Connecting to Cloud Foundry")

	client, err := GetEnvClient()
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("unable to create Cloud Foundry client: %v", err)
	}

	step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL))
	step.Done()

	org, space, err := selectOrgAndSpace(b, client, sg)
	if err != nil {
		return nil, err
	}
	deployment.OrganisationGUID = org.GUID
	deployment.SpaceGUID = space.GUID

	step = sg.Add(fmt.Sprintf("Searching app: %s", deployment.Name))
	apps, _, err := client.GetApplications(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{deployment.OrganisationGUID},
	}, ccv3.Query{
		Key:    ccv3.SpaceGUIDFilter,
		Values: []string{deployment.SpaceGUID},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{deployment.Name},
	})
	appExists := true
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to search for app: %v", err)
	}
	if len(apps) == 0 {
		appExists = false
	}

	if appExists {
		step.Update(fmt.Sprintf("Searching app: %s [found]", deployment.Name))
	} else {
		step.Update(fmt.Sprintf("Searching app: %s [not found]", deployment.Name))
	}
	step.Done()

	if appExists {
		app := apps[0]

		// Remove the app
		step = sg.Add(fmt.Sprintf("Deleting existing app %v (will be recreated)", deployment.Name))

		_, _, err = client.DeleteApplication(app.GUID)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("failed to delete app: %v", err)
		}

		step.Done()
	}

	// Create app
	step = sg.Add(fmt.Sprintf("Creating app %v", deployment.Name))
	appCreateRequest := resources.Application{
		Name:          deployment.Name,
		SpaceGUID:     space.GUID,
		LifecycleType: constant.AppLifecycleTypeDocker,
		State:         constant.ApplicationStarted,
	}

	app, _, err := client.CreateApplication(appCreateRequest)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to create app: %v", err)
	}
	step.Done()
	deployment.AppGUID = app.GUID

	if memoryMB != 0 || diskMB != 0 || instances > 1 {
		step = sg.Add("Configuring quota...")
		processes, _, err := client.GetApplicationProcesses(app.GUID)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("failed to get application processes: %v", err)
		}

		if len(processes) == 0 {
			step.Abort()
			return nil, fmt.Errorf("no processes found")
		}

		newP := ccv3.Process{
			Type: "web",
		}
		if memoryMB != 0 {
			newP.MemoryInMB = types.NullUint64{
				IsSet: true,
				Value: memoryMB,
			}
		}
		if diskMB != 0 {
			newP.DiskInMB = types.NullUint64{
				IsSet: true,
				Value: diskMB,
			}
		}

		if instances != 0 {
			newP.Instances = types.NullInt {
				IsSet: true,
				Value: int(instances),
			}
		}

		_, _, err = client.CreateApplicationProcessScale(app.GUID, newP)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("unable to scale application: %v", err)
		}
		step.Done()
	}

	// Create package
	step = sg.Add(fmt.Sprintf("Creating new package for docker image %s:%s in app", img.Image, img.Tag))

	dockerPackage := ccv3.Package{
		Type:        constant.PackageTypeDocker,
		DockerImage: fmt.Sprintf("%s:%s", img.Image, img.Tag),
		Relationships: resources.Relationships{
			constant.RelationshipTypeApplication: resources.Relationship{GUID: deployment.AppGUID},
		},
	}

	dockerUsername, dockerPassword, err := getDockerCredentialsFromEncodedAuth(b.config.DockerEncodedAuth)
	// only set docker credentials if they were provided correctly
	if err == nil {
		dockerPackage.DockerUsername = dockerUsername
		dockerPackage.DockerPassword = dockerPassword
	}

	cfPackage, _, err := client.CreatePackage(dockerPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %v", err)
	}
	step.Done()

	// Create build for package
	step = sg.Add(fmt.Sprintf("Creating a new build for the created package of image %v", cfPackage.DockerImage))
	cfBuild, _, err := client.CreateBuild(ccv3.Build{
		PackageGUID: cfPackage.GUID,
	})
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to create build: %v", err)
	}

	// Wait for droplet to become ready
	for {
		if cfBuild.State == constant.BuildStaged {
			break
		} else if cfBuild.State == constant.BuildFailed {
			step.Abort()
			return nil, fmt.Errorf("staging build failed: %v", cfBuild.Error)
		}
		time.Sleep(time.Second)
		cfBuild, _, _ = client.GetBuild(cfBuild.GUID)
		step.Update(fmt.Sprintf("Creating a new build for the created package of image %v [%v]", cfPackage.DockerImage, cfBuild.State))
	}
	step.Done()

	if len(b.config.Env) != 0 || b.config.EnvFromFile != "" {
		step = sg.Add("Assigning environment variables")
		envVars := ccv3.EnvironmentVariables{}

		// Precedence: envFromFile, env
		if b.config.EnvFromFile != "" {
			step := sg.Add("Adding environment variables from file")
			envContent := utils.ParseEnv(b.config.EnvFromFile)
			for k, v := range envContent {
				addFilteredEnvVar(envVars, k, v)
			}
			step.Done()
		}

		if len(b.config.Env) != 0 {
			step := sg.Add("Adding environment variables from HCL")
			for k, v := range b.config.Env {
				addFilteredEnvVar(envVars, k, v)
			}
			step.Done()
		}

		_, _, err = client.UpdateApplicationEnvironmentVariables(app.GUID, envVars)
		if err != nil {
			step.Abort()
			return nil, fmt.Errorf("unable to set environment variables: %v", err)
		}
		step.Done()
	}

	// Create deployment
	step = sg.Add("Creating a new deployment")
	_, _, err = client.CreateApplicationDeployment(deployment.AppGUID, cfBuild.DropletGUID)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to create deployment: %v", err)
	}
	step.Done()

	// Bind route
	routeUrl := fmt.Sprintf("%v.%v", deployment.Name, b.config.Domain)
	step = sg.Add(fmt.Sprintf("Binding route %v to application", routeUrl))
	domains, _, err := client.GetDomains(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{b.config.Domain},
	})
	if err != nil || len(domains) == 0 {
		step.Abort()
		return nil, fmt.Errorf("failed to get specified domain: %v", err)
	}
	domain := domains[0]

	route, _, err := client.CreateRoute(resources.Route{
		DomainGUID: domain.GUID,
		SpaceGUID:  deployment.SpaceGUID,
		Host:       deployment.Name,
		Destinations: []resources.RouteDestination{{
			App: resources.RouteDestinationApp{
				GUID: deployment.AppGUID,
			},
		}},
	})
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to create route: %v", err)
	}

	// Also map
	_, err = client.MapRoute(route.GUID, deployment.AppGUID)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to map route: %v", err)
	}
	step.Done()
	deployment.Url = route.URL

	return &deployment, nil
}

func parseQuantity(entry string) (uint64, error) {
	quantity, err := resource.ParseQuantity(entry)
	if err != nil {
		return 0, err
	}
	cv, fastConv := quantity.AsInt64()
	if fastConv == false {
		return 0, fmt.Errorf("fast conversion not available")
	}
	return uint64(cv) / 1024 / 1024, nil
}

func addFilteredEnvVar(envVars ccv3.EnvironmentVariables, k string, v string) {
	filteredString := types.NewFilteredString(v)
	if filteredString != nil {
		envVars[k] = *filteredString
	}
}
