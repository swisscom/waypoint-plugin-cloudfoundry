package cloudfoundry

import "code.cloudfoundry.org/cli/resources"

func (c *Client) GetServiceInstances(
	spaceGuid string,
	serviceName string,
) (resources.ServiceInstance, error) {
	serviceInstance, _, warns, err := c.client.GetServiceInstanceByNameAndSpace(serviceName, spaceGuid)
	c.listWarnings(warns)
	return serviceInstance, err
}
