package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"fmt"
)

func (c *Client) GetSpaceByName(spaceName string, orgGuid string) (space resources.Space, err error) {
	spaces, _, warns, err := c.client.GetSpaces(
		ccv3.Query{
			Key:    ccv3.OrganizationGUIDFilter,
			Values: []string{orgGuid},
		},
		ccv3.Query{
			Key:    ccv3.NameFilter,
			Values: []string{spaceName},
		})
	c.listWarnings(warns)
	if len(spaces) != 1 {
		return space, fmt.Errorf("expected 1 space, found %d", len(spaces))
	}
	return spaces[0], err
}
