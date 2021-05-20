package platform

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	ccWrapper "code.cloudfoundry.org/cli/api/cloudcontroller/wrapper"
	"code.cloudfoundry.org/cli/cf/api"
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

func GetEnvClient() (*ccv3.Client, error) {
	config, err := configv3.LoadConfig(configv3.FlagOverride{})
	if err != nil {
		return nil, err
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
	return ccClient, nil
}

type noPrinter struct {
	
}

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

func (n noPrinter) Print(v ...interface{}) {}

func (n noPrinter) Printf(format string, v ...interface{}) {}

func (n noPrinter) Println(v ...interface{}) {}

func (n noPrinter) WritesToConsole() bool {
	return false
}

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

	ui := terminal.NewUI(nil, nil, noTerminalPrinter{}, noPrinter{})
	controllerGateway := net.NewCloudControllerGateway(
		config,
		time.Now,
		ui,
		noPrinter{},
		"")

	return api.NewCloudControllerServiceBindingRepository(config, controllerGateway), nil
}