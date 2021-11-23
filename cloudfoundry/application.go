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

func filterQuery(query []ccv3.Query, queryKey ccv3.QueryKey, value string) []ccv3.Query {
	return append(query, ccv3.Query{Key: queryKey, Values: []string{value}})
}

func (c *Client) GetApplicationsByLabels(
	orgGuid string,
	spaceGuid string,
	labels []string,
) (apps []resources.Application, err error) {
	var warns ccv3.Warnings
	var query []ccv3.Query

	query = filterQuery(query, ccv3.OrganizationGUIDFilter, orgGuid)
	query = filterQuery(query, ccv3.SpaceGUIDFilter, spaceGuid)

	for _, label := range labels {
		query = filterQuery(query, ccv3.LabelSelectorFilter, label)
	}

	apps, warns, err = c.client.GetApplications(query...)

	c.listWarnings(warns)
	return apps, err
}

func (c *Client) StopApplication(guid string) (resources.Application, error) {
	app, warn, err := c.client.UpdateApplicationStop(guid)
	c.listWarnings(warn)
	return app, err
}

func (c *Client) DeleteApplication(guid string) (ccv3.JobURL, error) {
	jobUrl, warn, err := c.client.DeleteApplication(guid)
	c.listWarnings(warn)
	return jobUrl, err
}
