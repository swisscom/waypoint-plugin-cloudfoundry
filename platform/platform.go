package platform

import (
	"context"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/cli/types"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	proto "github.com/hashicorp/waypoint-plugin-sdk/proto/gen"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/cloudfoundry"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/utils"
	"k8s.io/apimachinery/pkg/api/resource"

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

type UserPasswordCredentials struct {
	Username string
	Password string
}

type QuotaConfig struct {
	Memory    string `hcl:"memory,optional"`
	Disk      string `hcl:"disk,optional"`
	Instances uint64 `hcl:"instances,optional"`
}

type HealthCheckConfig struct {
	Type              string `hcl:"type"`
	Endpoint          string `hcl:"endpoint,optional"`
	InvocationTimeout int64  `hcl:"invocation_timeout,optional"`
	Timeout           int64  `hcl:"timeout,optional"`
}

type DockerConfig struct {
	Username string `hcl:"username"`
}

type Config struct {
	Organisation             string             `hcl:"organisation"`
	Space                    string             `hcl:"space"`
	Docker                   *DockerConfig      `hcl:"docker,block"`
	Domain                   string             `hcl:"domain"`
	Quota                    *QuotaConfig       `hcl:"quota,block"`
	HealthCheck              *HealthCheckConfig `hcl:"health_check,block"`
	Env                      map[string]string  `hcl:"env,optional"`
	EnvFromFile              string             `hcl:"env_from_file,optional"`
	ServiceBindings          []string           `hcl:"service_bindings,optional"`
	DeploymentTimeoutSeconds string             `hcl:"deployment_timeout_seconds,optional"`
	deploymentTimeout        time.Duration
}

type Platform struct {
	config Config
	log    hclog.Logger
}

func (p *Platform) StatusFunc() interface{} {
	return p.Status
}

func (p *Platform) Status(
	ctx context.Context,
	log hclog.Logger,
	deployment *Deployment,
	ui terminal.UI,
) (*proto.StatusReport, error) {
	var result proto.StatusReport
	result.External = true

	sg := ui.StepGroup()
	defer sg.Wait()

	var err error

	step := sg.Add("Gathering health report for Cloud Foundry platform...")

	// Status of the Platform
	state := DeploymentState{}
	state.sg = &sg
	err = p.connectCloudFoundry(&state)
	if err != nil {
		ui.Output("Error connecting to Cloud Foundry: %s",
			err,
			terminal.WithErrorStyle(),
		)
		return nil, err
	}

	// Get processes by app
	theResult, err := state.client.GetHealthByGUID(deployment.AppGUID)

	step.Done()
	return theResult, nil
}

// Config implements Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// ConfigSet implements ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	_, ok := config.(*Config)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *Config as parameter")
	}

	return nil
}

// DeployFunc implements Builder
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.Deploy
}

type DeploymentState struct {
	deployment    *Deployment
	shouldCleanup bool

	sg                *terminal.StepGroup
	client            *cloudfoundry.Client
	space             *resources.Space
	org               *resources.Organization
	img               *docker.Image
	cfPackage         *resources.Package
	quotaParams       *QuotaParams
	healthCheckParams *HealthCheckParams
	app               *resources.Application
	appExists         bool
	apps              []resources.Application
	cfBuild           *resources.Build
	route             *resources.Route
	metadata          *resources.Metadata
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

func (p *Platform) Deploy(
	_ context.Context,
	log hclog.Logger,
	src *component.Source,
	img *docker.Image,
	_ *component.DeploymentConfig,
	ui terminal.UI,
) (*Deployment, error) {
	state := DeploymentState{
		img:        img,
		deployment: &Deployment{},
	}

	var err error
	state.deployment.Id = uuid.New().String()[:8]

	sg := ui.StepGroup()
	state.sg = &sg

	p.log = log

	// Parse quantities
	err = p.validateQuota(&state)
	if err != nil {
		return nil, err
	}

	err = p.validateHealthCheck(&state)
	if err != nil {
		return nil, err
	}

	// Validate timeout value
	err = p.parseTimeout()
	if err != nil {
		return nil, err
	}

	appName := src.App
	log.Debug("deployment name generation", "deployment", state.deployment.Id, "appName", appName)
	state.deployment.Name = fmt.Sprintf("%v-%v", appName, state.deployment.Id)

	err = p.connectCloudFoundry(&state)
	if err != nil {
		return nil, err
	}

	org, space, err := state.client.SelectOrgAndSpace(p.config.Organisation, p.config.Space)
	if err != nil {
		return nil, err
	}

	state.space = &space
	state.org = &org

	state.deployment.OrganisationGUID = org.GUID
	state.deployment.SpaceGUID = space.GUID

	err = p.searchApp(&state)
	if err != nil {
		return nil, err
	}

	err = p.deleteApp(&state, 0)
	if err != nil {
		return nil, err
	}

	// Adding label with app name, useful for searching of instances
	state.metadata = &resources.Metadata{
		Labels: map[string]types.NullString{"appName": {Value: src.App, IsSet: true}},
	}

	state.app, err = p.createApp(&state)
	state.shouldCleanup = true
	defer p.cleanupResourcesOnFail(&state)

	if err != nil {
		return nil, err
	}
	state.deployment.AppGUID = state.app.GUID

	err = p.configureQuota(state)
	if err != nil {
		return nil, err
	}

	state.cfPackage, err = p.createPackage(state)
	if err != nil {
		return nil, err
	}

	err = p.setEnvironmentVariables(&state)
	if err != nil {
		return nil, err
	}

	err = p.bindServices(&state)
	if err != nil {
		return nil, err
	}

	err = p.createBuild(&state)
	if err != nil {
		return nil, err
	}

	err = p.createDeployment(&state)
	if err != nil {
		return nil, err
	}

	err = p.waitProcess(&state)
	if err != nil {
		return nil, err
	}

	err = p.configureHealthCheck(&state)
	if err != nil {
		return nil, err
	}

	err = p.bindRoute(&state)
	if err != nil {
		return nil, err
	}

	state.shouldCleanup = false

	log.Debug("route_url", "url", state.route.URL)
	state.deployment.Url = state.route.URL

	return state.deployment, nil
}

func (p *Platform) createApp(state *DeploymentState) (*resources.Application, error) {
	step := (*state.sg).Add(fmt.Sprintf("Creating app %v", state.deployment.Name))
	appCreateRequest := resources.Application{
		Name:          state.deployment.Name,
		SpaceGUID:     state.space.GUID,
		LifecycleType: constant.AppLifecycleTypeDocker,
		State:         constant.ApplicationStarted,
		Metadata:      state.metadata,
	}

	app, _, err := state.client.CfClient().CreateApplication(appCreateRequest)
	if err != nil {
		step.Abort()
		return nil, fmt.Errorf("failed to create app: %v", err)
	}
	step.Done()
	return &app, nil
}

func (p *Platform) Generation() ([]byte, error) {
	return uuid.New().MarshalBinary()
}

// GenerationFunc implements component.Generation
func (p *Platform) GenerationFunc() interface{} {
	return p.Generation
}

func (p *Platform) createPackage(state DeploymentState) (*resources.Package, error) {
	step := (*state.sg).Add(fmt.Sprintf("Creating new package for docker image %s:%s in app",
		state.img.Image,
		state.img.Tag,
	))
	dockerPackage := resources.Package{
		Type:        constant.PackageTypeDocker,
		DockerImage: fmt.Sprintf("%s:%s", state.img.Image, state.img.Tag),
		Relationships: resources.Relationships{
			constant.RelationshipTypeApplication: resources.Relationship{GUID: state.deployment.AppGUID},
		},
	}

	// Docker pull credentials
	if p.config.Docker != nil {
		// Using docker credentials
		dockerPackage.DockerUsername = p.config.Docker.Username

		// Docker password from env variable
		dockerPackage.DockerPassword = os.Getenv("CF_DOCKER_PASSWORD")
		if dockerPackage.DockerPassword == "" {
			step.Abort()
			return nil, fmt.Errorf("invalid docker credentials: %s", errDockerPasswordEmpty)
		}
	}

	cfPackage, err := state.client.CreatePackage(dockerPackage)
	if err != nil {
		return nil, fmt.Errorf("failed to create package: %v", err)
	}
	step.Done()

	return &cfPackage, nil
}

func (p *Platform) connectCloudFoundry(state *DeploymentState) error {
	step := (*state.sg).Add("Connecting to Cloud Foundry")
	client, err := cloudfoundry.New(p.log)
	if err != nil {
		step.Abort()
		return fmt.Errorf("unable to create Cloud Foundry client: %v", err)
	}

	state.client = client

	step.Update(fmt.Sprintf("Connecting to Cloud Foundry at %s", client.CloudControllerURL()))
	step.Done()
	return nil
}

func (p *Platform) configureQuota(state DeploymentState) error {
	if state.quotaParams.memoryMb != 0 || state.quotaParams.diskMb != 0 || state.quotaParams.instances > 1 {
		step := (*state.sg).Add("Configuring quota...")
		processes, err := state.client.GetApplicationProcesses(state.app.GUID)
		if err != nil {
			step.Abort()
			return fmt.Errorf("failed to get application processes: %v", err)
		}

		if len(processes) == 0 {
			step.Abort()
			return fmt.Errorf("no processes found")
		}

		newP := resources.Process{
			Type: "web",
		}
		if state.quotaParams.memoryMb != 0 {
			newP.MemoryInMB = types.NullUint64{
				IsSet: true,
				Value: state.quotaParams.memoryMb,
			}
		}
		if state.quotaParams.diskMb != 0 {
			newP.DiskInMB = types.NullUint64{
				IsSet: true,
				Value: state.quotaParams.diskMb,
			}
		}

		if state.quotaParams.instances != 0 {
			newP.Instances = types.NullInt{
				IsSet: true,
				Value: int(state.quotaParams.instances),
			}
		}

		_, err = state.client.CreateApplicationProcessScale(state.app.GUID, newP)
		if err != nil {
			step.Abort()
			return fmt.Errorf("unable to scale application: %v", err)
		}
		step.Done()
	}
	return nil
}

func (p *Platform) configureHealthCheck(state *DeploymentState) error {
	if needToConfigureHealthCheck(state) {
		step := (*state.sg).Add("Configuring health check...")

		processes, err := state.client.GetApplicationProcesses(state.app.GUID)
		if err != nil {
			step.Abort()
			return fmt.Errorf("failed to get application processes: %v", err)
		}

		if len(processes) == 0 {
			step.Abort()
			return fmt.Errorf("no processes found")
		}

		for _, process := range processes {
			process.HealthCheckType = state.healthCheckParams.Type

			if state.healthCheckParams.Endpoint != "" {
				process.HealthCheckEndpoint = state.healthCheckParams.Endpoint
			}

			if state.healthCheckParams.InvocationTimeout != 0 {
				process.HealthCheckInvocationTimeout = state.healthCheckParams.InvocationTimeout
			}

			if state.healthCheckParams.Timeout != 0 {
				process.HealthCheckTimeout = state.healthCheckParams.Timeout
			}

			p.log.Debug("updating process: %v", process)

			_, err = state.client.UpdateApplicationProcess(process)
			if err != nil {
				step.Abort()
				return fmt.Errorf("unable to configure application process: %v", err)
			}
		}

		step.Done()
	}
	return nil
}

func needToConfigureHealthCheck(state *DeploymentState) bool {
	return state.healthCheckParams != nil && (state.healthCheckParams.Endpoint != "" ||
		state.healthCheckParams.Type != "" || state.healthCheckParams.InvocationTimeout != 0 ||
		state.healthCheckParams.Timeout != 0)
}

type QuotaParams struct {
	diskMb    uint64
	memoryMb  uint64
	instances uint64
}

type HealthCheckParams struct {
	Type              constant.HealthCheckType
	Endpoint          string
	InvocationTimeout int64
	Timeout           int64
}

func (p *Platform) validateQuota(state *DeploymentState) error {
	var err error

	p.log.Debug("validate quota")

	step := (*state.sg).Add("Validating quota parameters")
	state.quotaParams = &QuotaParams{}
	if p.config.Quota != nil {
		p.log.Debug("quota is not nil")
		if p.config.Quota.Instances > 0 {
			state.quotaParams.instances = p.config.Quota.Instances
		}
		if p.config.Quota.Memory != "" {
			p.log.Debug("quota memory", "quota_memory", p.config.Quota.Memory)
			state.quotaParams.memoryMb, err = parseQuantity(p.config.Quota.Memory)
			if err != nil {
				step.Abort()
				return fmt.Errorf("unable to parse memory: %v", err)
			}
			p.log.Debug("quota parsed", "quota_parsed", state.quotaParams.memoryMb)
		}
		if p.config.Quota.Disk != "" {
			p.log.Debug("quota disk", "quota_disk", p.config.Quota.Disk)
			state.quotaParams.diskMb, err = parseQuantity(p.config.Quota.Disk)
			if err != nil {
				return fmt.Errorf("unable to parse disk: %v", err)
			}
			p.log.Debug("quota parsed disk", "quota_parsed", state.quotaParams.diskMb)
		}
	}
	step.Done()
	return nil
}

func (p *Platform) validateHealthCheck(state *DeploymentState) error {
	p.log.Debug("validate health check")

	step := (*state.sg).Add("Validating health check parameters")
	state.healthCheckParams = &HealthCheckParams{}

	if p.config.HealthCheck != nil {
		p.log.Debug("health check config is not nil")

		state.healthCheckParams.Type = constant.HealthCheckType(p.config.HealthCheck.Type)

		if state.healthCheckParams.Type == constant.HTTP && p.config.HealthCheck.Endpoint == "" {
			step.Abort()
			return fmt.Errorf("undefined endpoint for HTTP health check")
		}
		state.healthCheckParams.Endpoint = p.config.HealthCheck.Endpoint

		if p.config.HealthCheck.InvocationTimeout < 0 || p.config.HealthCheck.InvocationTimeout > 180 {
			step.Abort()
			return fmt.Errorf("invocation timeout has to be 0-180s")
		}
		state.healthCheckParams.InvocationTimeout = p.config.HealthCheck.InvocationTimeout

		if p.config.HealthCheck.Timeout < 0 || p.config.HealthCheck.Timeout > 180 {
			step.Abort()
			return fmt.Errorf("timeout has to be 0-180s")
		}
		state.healthCheckParams.Timeout = p.config.HealthCheck.Timeout

		p.log.Debug("health check params: %v", state.healthCheckParams)

	}

	step.Done()
	return nil
}

func (p *Platform) searchApp(state *DeploymentState) error {
	step := (*state.sg).Add(fmt.Sprintf("Searching app: %s", state.deployment.Name))
	var err error
	state.apps, err = state.client.GetApplications(
		state.deployment.OrganisationGUID,
		state.deployment.SpaceGUID,
		state.deployment.Name,
	)
	state.appExists = true
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to search for app: %v", err)
	}
	if len(state.apps) == 0 {
		state.appExists = false
	}

	if state.appExists {
		step.Update(fmt.Sprintf("Searching app: %s [found]", state.deployment.Name))
	} else {
		step.Update(fmt.Sprintf("Searching app: %s [not found]", state.deployment.Name))
	}
	step.Done()
	return nil
}

func (p *Platform) deleteApp(state *DeploymentState, idx int) error {
	if state.appExists {
		if idx >= len(state.apps) {
			return fmt.Errorf("unable to delete app index %d when only %d apps are available",
				idx,
				len(state.apps),
			)
		}

		app := state.apps[idx]

		// Remove the app
		step := (*state.sg).Add(fmt.Sprintf(
			"Deleting existing app %v (will be recreated)",
			state.deployment.Name),
		)

		_, err := state.client.DeleteApplication(app.GUID)
		if err != nil {
			step.Abort()
			return fmt.Errorf("failed to delete app: %v", err)
		}
		step.Done()
	}
	return nil
}

func (p *Platform) createBuild(state *DeploymentState) error {
	// Create build for package
	step := (*state.sg).Add(
		fmt.Sprintf("Creating a new build for the created package of image %v",
			state.cfPackage.DockerImage))

	cfBuild, err := state.client.CreateBuild(state.cfPackage.GUID)
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to create build: %v", err)
	}

	state.cfBuild = &cfBuild

	p.log.Debug("build created", "cfbuild", fmt.Sprintf("%+v", cfBuild))

	// Wait for droplet to become ready
	for {
		if state.cfBuild.State == constant.BuildStaged {
			break
		} else if state.cfBuild.State == constant.BuildFailed || state.cfBuild.Error != "" {
			step.Abort()
			return fmt.Errorf("staging build failed: %v", state.cfBuild.Error)
		}

		time.Sleep(500 * time.Millisecond)
		build, _ := state.client.GetBuild(state.cfBuild.GUID)
		p.log.Debug("build update", "build", fmt.Sprintf("%+v", build))
		state.cfBuild = &build
		step.Update(
			fmt.Sprintf("Creating a new build for the created package of image %v [%v]",
				state.cfPackage.DockerImage,
				state.cfBuild.State),
		)
	}
	step.Done()
	return nil
}

func (p *Platform) createDeployment(state *DeploymentState) error {
	// Create deployment
	step := (*state.sg).Add("Creating a new deployment")
	_, err := state.client.CreateApplicationDeployment(state.deployment.AppGUID, state.cfBuild.DropletGUID)
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to create deployment: %v", err)
	}
	step.Done()
	return nil
}

func (p *Platform) bindRoute(state *DeploymentState) error {
	routeUrl := fmt.Sprintf("%v.%v", state.deployment.Name, p.config.Domain)
	step := (*state.sg).Add(fmt.Sprintf("Binding route %v to application", routeUrl))
	domains, err := state.client.GetDomains(p.config.Domain)
	if err != nil || len(domains) == 0 {
		step.Abort()
		return fmt.Errorf("failed to get specified domain: %v", err)
	}
	domain := domains[0]

	if len(p.config.ServiceBindings) > 0 {

	}

	route, err := state.client.CreateRoute(resources.Route{
		DomainGUID: domain.GUID,
		SpaceGUID:  state.deployment.SpaceGUID,
		Host:       state.deployment.Name,
		Destinations: []resources.RouteDestination{{
			App: resources.RouteDestinationApp{
				GUID: state.deployment.AppGUID,
			},
		}},
	})

	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to create route: %v", err)
	}

	state.route = &route

	// Also map
	err = state.client.MapRoute(route.GUID, state.deployment.AppGUID)
	if err != nil {
		step.Abort()
		return fmt.Errorf("failed to map route: %v", err)
	}
	step.Done()
	return nil
}

func (p *Platform) setEnvironmentVariables(state *DeploymentState) error {
	// Set environment variables to app
	if len(p.config.Env) != 0 || p.config.EnvFromFile != "" {
		step := (*state.sg).Add("Assigning environment variables")
		envVars := resources.EnvironmentVariables{}

		// Precedence: envFromFile, env
		if p.config.EnvFromFile != "" {
			step := (*state.sg).Add("Adding environment variables from file")
			envContent := utils.ParseEnv(p.config.EnvFromFile)
			for k, v := range envContent {
				addFilteredEnvVar(envVars, k, v)
			}
			step.Done()
		}

		if len(p.config.Env) != 0 {
			step := (*state.sg).Add("Adding environment variables from HCL")
			for k, v := range p.config.Env {
				addFilteredEnvVar(envVars, k, v)
			}
			step.Done()
		}

		_, err := state.client.UpdateApplicationEnvironmentVariables(state.app.GUID, envVars)
		if err != nil {
			step.Abort()
			return fmt.Errorf("unable to set environment variables: %v", err)
		}
		step.Done()
	}
	return nil
}

func (p *Platform) bindServices(state *DeploymentState) error {
	if len(p.config.ServiceBindings) > 0 {
		step := (*state.sg).Add("Binding services")
		// get ServiceBind Repository
		sbRepo, err := cloudfoundry.GetServiceBindRepository()
		if err != nil {
			step.Abort()
			return fmt.Errorf("unable to get ServiceBind Repository: %v", err)
		}

		for _, serviceName := range p.config.ServiceBindings {
			// find service
			serviceInstance, err := state.client.GetServiceInstances(state.deployment.SpaceGUID, serviceName)
			if err != nil {
				step.Abort()
				return fmt.Errorf("unable to get service %s: %v", serviceName, err)
			}

			// bind service
			p.log.Debug("create service binding",
				"serviceInstance", serviceInstance,
				"app", state.app,
				"serviceInstanceGUID", serviceInstance.GUID,
				"appGUID", state.app.GUID,
			)
			err = sbRepo.Create(serviceInstance.GUID, state.app.GUID, map[string]interface{}{})
			if err != nil {
				step.Abort()
				return fmt.Errorf("unable to bind service %s to app: %v", serviceName, err)
			}
		}
		step.Done()
	}
	return nil
}

func (p *Platform) cleanupResourcesOnFail(state *DeploymentState) {
	if !state.shouldCleanup {
		return
	}

	// Clean up resources that were created but failed
	if state.deployment != nil {
		_, err := state.client.DeleteApplication(state.deployment.AppGUID)
		if err != nil {
			p.log.Error("unable to delete application",
				"guid", state.deployment.AppGUID,
				"error", err,
			)
			return
		}
	}
}

func (p *Platform) waitProcess(state *DeploymentState) error {
	if state.deployment == nil {
		return fmt.Errorf("unable to wait for a nil deployment")
	}

	applicationProcesses, err := state.client.GetApplicationProcesses(state.deployment.AppGUID)
	if err != nil {
		return fmt.Errorf("unable to get application processes: %v", err)
	}

	startTime := time.Now()
	for {
		processes := map[resources.Process][]ccv3.ProcessInstance{}
		if time.Now().After(startTime.Add(p.config.deploymentTimeout)) {
			return fmt.Errorf(
				"timeout: %.0f seconds elapsed but the application isn't started yet. "+
					"deployment=%v, processes=%v",
				p.config.deploymentTimeout.Seconds(),
				state.deployment,
				processes,
			)
		}

		for _, proc := range applicationProcesses {
			procInstances, err := state.client.GetProcessInstances(proc.GUID)
			if err != nil {
				return fmt.Errorf("unable to get process instances for process %v", proc)
			}
			processes[proc] = procInstances
		}

		starting := false

		// Check status
		for process, processInstances := range processes {
			for _, instance := range processInstances {
				switch instance.State {
				case constant.ProcessInstanceCrashed:
					return fmt.Errorf("deployment failed: process crashed, process=%v, processInstance=%v",
						process, instance,
					)
				case constant.ProcessInstanceStarting:
					starting = true
				}
			}
		}

		if !starting {
			p.log.Info("processes are ready!", "processes", processes)
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func (p *Platform) parseTimeout() error {
	if p.config.DeploymentTimeoutSeconds == "" {
		// User didn't provide any timeout, using the default
		p.config.deploymentTimeout = DefaultDeploymentTimeout
		return nil
	}

	d, err := time.ParseDuration(p.config.DeploymentTimeoutSeconds)
	if err != nil {
		return err
	}

	if d < 0 {
		return fmt.Errorf("duration cannot be negative")
	}

	if d == 0 {
		p.log.Warn("duration cannot be 0, resetting to default")
		p.config.deploymentTimeout = DefaultDeploymentTimeout
	} else {
		p.config.deploymentTimeout = d
	}

	return nil
}

func (p *Platform) getOrganizationByName(org string, state *DeploymentState) (*resources.Organization, error) {
	organization, err := state.client.GetOrganization(p.config.Organisation)
	if err != nil {
		return nil, err
	}
	return &organization, nil
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

func addFilteredEnvVar(envVars resources.EnvironmentVariables, k string, v string) {
	filteredString := types.NewFilteredString(v)
	if filteredString != nil {
		envVars[k] = *filteredString
	}
}

var _ component.Platform = (*Platform)(nil)
var _ component.Generation = (*Platform)(nil)
var _ component.Status = (*Platform)(nil)
