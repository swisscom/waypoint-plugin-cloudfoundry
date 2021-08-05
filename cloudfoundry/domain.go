package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
)

func (c *Client) GetDomains(domains ...string) ([]resources.Domain, error) {
	domainsObj, warns, err := c.client.GetDomains(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: domains,
	})
	c.listWarnings(warns)
	return domainsObj, err
}
