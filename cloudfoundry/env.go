package cloudfoundry

import "code.cloudfoundry.org/cli/resources"

func (c *Client) UpdateApplicationEnvironmentVariables(
	appGuid string,
	vars resources.EnvironmentVariables,
) (resources.EnvironmentVariables, error) {
	envVars, warns, err := c.client.UpdateApplicationEnvironmentVariables(appGuid, vars)
	c.listWarnings(warns)
	return envVars, err
}