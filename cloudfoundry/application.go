package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
)

func (c *Client) GetApplications(
	orgGuid string,
	spaceGuid string,
	appName string,
) (apps []resources.Application, err error) {
	var warns ccv3.Warnings
	apps, warns, err = c.client.GetApplications(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGuid},
	}, ccv3.Query{
		Key:    ccv3.SpaceGUIDFilter,
		Values: []string{spaceGuid},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{appName},
	})

	c.listWarnings(warns)
	return apps, err
}

func (c *Client) DeleteApplication(guid string) (ccv3.JobURL, error) {
	jobUrl, warn, err := c.client.DeleteApplication(guid)
	c.listWarnings(warn)
	return jobUrl, err
}