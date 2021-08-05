package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"fmt"
)

func (c *Client) SelectOrgAndSpace(
	orgName string,
	spaceName string,
) (org resources.Organization, space resources.Space, err error) {
	organizations, _, err := c.client.GetOrganizations(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{orgName},
	})
	if err != nil || len(organizations) != 1 {
		return org, space, fmt.Errorf("failed to select organisation: %v", err)
	}
	org = organizations[0]

	spaces, _, _, err := c.client.GetSpaces(ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{org.GUID},
	}, ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{spaceName},
	})
	if err != nil || len(spaces) != 1 {
		return org, space, fmt.Errorf("failed to select space: %v", err)
	}

	space = spaces[0]
	return org, space, nil
}
