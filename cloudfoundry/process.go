package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
)

func (c *Client) GetApplicationProcesses(appGuid string) (processes []resources.Process, err error) {
	var warns ccv3.Warnings
	processes, warns, err = c.client.GetApplicationProcesses(appGuid)
	c.listWarnings(warns)
	return processes, err
}

func (c *Client) CreateApplicationProcessScale(guid string, p resources.Process) (proc resources.Process, err error) {
	var warns ccv3.Warnings
	proc, warns, err = c.client.CreateApplicationProcessScale(guid, p)
	c.listWarnings(warns)
	return proc, err
}