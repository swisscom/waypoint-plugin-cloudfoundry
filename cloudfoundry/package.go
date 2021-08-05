package cloudfoundry

import "code.cloudfoundry.org/cli/resources"

func (c *Client) CreatePackage(pkg resources.Package) (resources.Package, error) {
	p, warns, err := c.client.CreatePackage(pkg)
	c.listWarnings(warns)
	return p, err
}
