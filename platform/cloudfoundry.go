package platform

import (
	"bytes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	ccWrapper "code.cloudfoundry.org/cli/api/cloudcontroller/wrapper"
	"code.cloudfoundry.org/cli/api/uaa"
	uaaWrapper "code.cloudfoundry.org/cli/api/uaa/wrapper"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/commandregistry"
	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/cli/resources"
	"code.cloudfoundry.org/cli/util/configv3"
	"fmt"
	wpTerm "github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"time"
)

func selectOrgAndSpace(b *Platform, client *ccv3.Client, sg wpTerm.StepGroup) (resources.Organization, resources.Space, error) {
	var org resources.Organization
	var space resources.Space

	step := sg.Add(fmt.Sprintf("Selecting organisation: %s", b.config.Organisation))
	orgs, _, err := client.GetOrganizations(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{b.config.Organisation},
	})
	if err != nil || len(orgs) != 1 {
		step.Abort()
		return org, space, fmt.Errorf("failed to select organisation: %v", err)
	}
	org = orgs[0]
	step.Done()

	step = sg.Add(fmt.Sprintf("Selecting space: %s", b.config.Space))
	spaces, _, _, err := client.GetSpaces(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{org.GUID},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{b.config.Space},
	})
	if err != nil || len(spaces) != 1 {
		step.Abort()
		return org, space, fmt.Errorf("failed to select space: %v", err)
	}
	space = spaces[0]
	step.Done()

	return org, space, nil
}

func getEnvClientConfig() (*ccv3.Client, *configv3.Config, error) {
	config, err := configv3.LoadConfig(configv3.FlagOverride{})
	if err != nil {
		return nil, nil, err
	}

	var ccWrappers []ccv3.ConnectionWrapper
	authWrapper := ccWrapper.NewUAAAuthentication(nil, config)

	ccWrappers = append(ccWrappers, authWrapper)
	ccWrappers = append(ccWrappers, ccWrapper.NewRetryRequest(config.RequestRetryCount()))

	ccClient := ccv3.NewClient(ccv3.Config{
		AppName:            config.BinaryName(),
		AppVersion:         config.BinaryVersion(),
		JobPollingTimeout:  config.OverallPollingTimeout(),
		JobPollingInterval: config.PollingInterval(),
		Wrappers:           ccWrappers,
	})

	ccClient.TargetCF(ccv3.TargetSettings{
		URL:               config.Target(),
		SkipSSLValidation: config.SkipSSLValidation(),
		DialTimeout:       config.DialTimeout(),
	})

	uaaClient := uaa.NewClient(config)
	uaaAuthWrapper := uaaWrapper.NewUAAAuthentication(nil, config)
	uaaClient.WrapConnection(uaaAuthWrapper)
	uaaClient.WrapConnection(uaaWrapper.NewRetryRequest(config.RequestRetryCount()))

	err = uaaClient.SetupResources(config.ConfigFile.UAAEndpoint, config.ConfigFile.AuthorizationEndpoint)
	if err != nil {
		return nil, nil, err
	}

	uaaAuthWrapper.SetClient(uaaClient)
	authWrapper.SetClient(uaaClient)
	return ccClient, config, nil
}

func GetEnvClient() (*ccv3.Client, error) {
	client, _, err := getEnvClientConfig()
	return client, err
}

func GetServiceBindRepository() (api.ServiceBindingRepository, error) {
	configPath, err := confighelpers.DefaultFilePath()
	if err != nil {
		return nil, err
	}

	var theError error
	errHandler := func(err error) {
		theError = err
	}
	config := coreconfig.NewRepositoryFromFilepath(configPath, errHandler)
	if theError != nil {
		return nil, theError
	}

	var writerBuffer []byte
	logger := noPrinter{}

	envDialTimeout := ""
	deps := commandregistry.NewDependency(bytes.NewBuffer(writerBuffer), logger, envDialTimeout)
	defer deps.Config.Close()

	uaaGateway := net.NewUAAGateway(deps.Config, deps.UI, logger, envDialTimeout)
	cloudController := net.NewCloudControllerGateway(deps.Config, time.Now, deps.UI, logger, envDialTimeout)

	deps.Gateways = map[string]net.Gateway{
		"cloud-controller": cloudController,
		"uaa":              uaaGateway,
		"routing-api":      net.NewRoutingAPIGateway(deps.Config, time.Now, deps.UI, logger, envDialTimeout),
	}

	uaaRepository := authentication.NewUAARepository(uaaGateway, config, net.NewRequestDumper(noPrinter{}))
	cloudController.SetTokenRefresher(uaaRepository)

	return api.NewCloudControllerServiceBindingRepository(config, cloudController), nil
}
