package cloudfoundry

import "code.cloudfoundry.org/cli/resources"

func (c *Client) CreateBuild(packageGuid string) (resources.Build, error) {
	build, warns, err := c.client.CreateBuild(resources.Build{
		PackageGUID: packageGuid,
	})
	c.listWarnings(warns)
	return build, err
}

func (c *Client) GetBuild(buildGuid string) (resources.Build, error) {
	build, warns, err := c.client.GetBuild(buildGuid)
	c.listWarnings(warns)
	return build, err
}
