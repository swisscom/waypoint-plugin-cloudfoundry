package platform

import (
	"encoding/base64"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	ccWrapper "code.cloudfoundry.org/cli/api/cloudcontroller/wrapper"
	"code.cloudfoundry.org/cli/api/uaa"
	uaaWrapper "code.cloudfoundry.org/cli/api/uaa/wrapper"
	"code.cloudfoundry.org/cli/resources"
	"code.cloudfoundry.org/cli/util/configv3"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

var UserAgent = "waypoint-plugin-cloudfoundry/v" + Version

func selectOrgAndSpace(b *Platform, client *ccv3.Client, sg terminal.StepGroup) (resources.Organization, resources.Space, error) {
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

	ccWrappers := []ccv3.ConnectionWrapper{}
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

	_, _, err = ccClient.TargetCF(ccv3.TargetSettings{
		URL:               config.Target(),
		SkipSSLValidation: config.SkipSSLValidation(),
		DialTimeout:       config.DialTimeout(),
	})
	if err != nil {
		return nil, err
	}

	uaaClient := uaa.NewClient(config)

	uaaAuthWrapper := uaaWrapper.NewUAAAuthentication(nil, config)
	uaaClient.WrapConnection(uaaAuthWrapper)
	uaaClient.WrapConnection(uaaWrapper.NewRetryRequest(config.RequestRetryCount()))

	err = uaaClient.SetupResources(config.ConfigFile.AuthorizationEndpoint)
	if err != nil {
		return nil, err
	}

	uaaAuthWrapper.SetClient(uaaClient)
	authWrapper.SetClient(uaaClient)
	return ccClient, nil
}

func getDockerCredentialsFromEncodedAuth(encodedAuth string) (string, string, error) {
	username, password, err := parseEncodedAuth(encodedAuth, ":")
	if err != nil {
		return "", "", fmt.Errorf("invalid auth data: %v", err)
	}

	return username, password, nil
}

func parseEncodedAuth(encodedAuth string, sep string) (string, string, error) {
	decodedCreds, err := base64.StdEncoding.DecodeString(encodedAuth)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 string")
	}

	split := strings.Split(string(decodedCreds), sep)
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid format")
	}

	return split[0], split[1], nil
}
