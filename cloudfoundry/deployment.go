package cloudfoundry

func (c *Client) CreateApplicationDeployment(appGuid string, dropletGuid string) (string, error) {
	guid, warns, err := c.client.CreateApplicationDeployment(appGuid, dropletGuid)
	c.listWarnings(warns)
	return guid, err
}
