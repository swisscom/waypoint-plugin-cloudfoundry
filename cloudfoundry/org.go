package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"fmt"
)

func (c *Client) GetOrganization(name string) (resources.Organization, error) {
	var org resources.Organization
	organizations, warns, err := c.client.GetOrganizations(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{name},
	})
	c.listWarnings(warns)
	if err != nil {
		return org, err
	}

	if len(organizations) != 1 {
		return org, fmt.Errorf("expected 1 org, found %d", len(organizations))
	}
	return org, nil
}
