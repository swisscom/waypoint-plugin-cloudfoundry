package cloudfoundry

import (
	"bytes"
	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/authentication"
	"code.cloudfoundry.org/cli/cf/commandregistry"
	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
	"time"
)

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
