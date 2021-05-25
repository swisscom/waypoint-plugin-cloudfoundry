package platform

import (
	"bytes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	ccWrapper "code.cloudfoundry.org/cli/api/cloudcontroller/wrapper"
	"code.cloudfoundry.org/cli/api/uaa"
	uaaWrapper "code.cloudfoundry.org/cli/api/uaa/wrapper"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/commandregistry"
	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"code.cloudfoundry.org/cli/resources"
	"code.cloudfoundry.org/cli/util/configv3"
	"fmt"
	wpTerm "github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"time"
)

var UserAgent = "waypoint-plugin-cloudfoundry/v" + Version

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

type stdoutPrinter struct {
	
}
type noPrinter struct {

}

func (n noPrinter) Print(v ...interface{}) {}

func (n noPrinter) Printf(format string, v ...interface{}) {}

func (n noPrinter) Println(v ...interface{}) {}

func (n noPrinter) WritesToConsole() bool { return false }

type noTerminalPrinter struct {

}

func (t noTerminalPrinter) Print(a ...interface{}) (n int, err error) {
	return 0, nil
}

func (t noTerminalPrinter) Printf(format string, a ...interface{}) (n int, err error) {
	return 0, nil
}

func (t noTerminalPrinter) Println(a ...interface{}) (n int, err error) {
	return 0, nil
}

func (n stdoutPrinter) Print(v ...interface{}) {
	fmt.Print(v...)
}

func (n stdoutPrinter) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (n stdoutPrinter) Println(v ...interface{}) {
	fmt.Println()
}

func (n stdoutPrinter) WritesToConsole() bool {
	return true
}

var _ trace.Printer = stdoutPrinter{}
var _ trace.Printer = noPrinter{}

var _ terminal.Printer = noTerminalPrinter{}

func GetServiceBindRepository() (api.ServiceBindingRepository, error) {
	configPath, err := confighelpers.DefaultFilePath()
	if err != nil {
		return nil, err
	}

	var theError error
	errHandler := func(err error){
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
	deps.Gateways = map[string]net.Gateway{
		"cloud-controller": net.NewCloudControllerGateway(deps.Config, time.Now, deps.UI, logger, envDialTimeout),
		"uaa":              net.NewUAAGateway(deps.Config, deps.UI, logger, envDialTimeout),
		"routing-api":      net.NewRoutingAPIGateway(deps.Config, time.Now, deps.UI, logger, envDialTimeout),
	}

	return api.NewCloudControllerServiceBindingRepository(config, deps.Gateways["cloud-controller"]), nil
}