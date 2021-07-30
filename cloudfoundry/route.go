package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"fmt"
)

func (c *Client) GetRoute(guid string) (route resources.Route, err error) {
	routes, warns, err := c.client.GetRoutes(
		ccv3.Query{
			Key: ccv3.RouteGUIDFilter,
			Values: []string{guid},
		},
	)
	c.listWarnings(warns)
	if err != nil {
		return route, err
	}
	if len(routes) != 1 {
		return route, fmt.Errorf("expected 1 route, got %d routes instead", len(routes))
	}
	route = routes[0]
	return route, nil
}

func (c *Client) UpsertRoute(
	hostname string,
	domain resources.Domain,
	spaceGuid string,
) (route resources.Route, err error) {
	routes, _, err := c.client.GetRoutes(ccv3.Query{
		Key:    ccv3.DomainGUIDFilter,
		Values: []string{domain.GUID},
	}, ccv3.Query{
		Key:    ccv3.HostsFilter,
		Values: []string{hostname},
	})

	if err != nil {
		return route, err
	}

	if len(routes) > 1 {
		return route, fmt.Errorf("more than one route returned")
	}

	if len(routes) == 1 {
		route = routes[0]
		return route, nil
	}

	route, err = c.CreateRoute(resources.Route{
		DomainGUID: domain.GUID,
		SpaceGUID:  spaceGuid,
		Host:       hostname,
	})

	if err != nil {
		return route, err
	}

	return route, nil
}

func (c *Client) MapRoute(routeGuid string, appGuid string) error {
	warns, err := c.client.MapRoute(routeGuid, appGuid)
	c.listWarnings(warns)
	return err
}

func (c *Client) UnmapRoute(routeGuid string, appGuid string) error {
	warns, err := c.client.UnmapRoute(routeGuid, appGuid)
	c.listWarnings(warns)
	return err
}

func (c *Client) CreateRoute(route resources.Route) (resources.Route, error) {
	route, warns, err := c.client.CreateRoute(route)
	c.listWarnings(warns)
	return route, err
}

func (c *Client) DeleteRoute(guid string) (ccv3.JobURL, error) {
	jobURL, warns, err := c.client.DeleteRoute(guid)
	c.listWarnings(warns)
	return jobURL, err
}

func (c *Client) GetApplicationRoutes(guid string) ([]resources.Route, error) {
	routes, warns, err := c.client.GetApplicationRoutes(guid)
	c.listWarnings(warns)
	return routes, err
}