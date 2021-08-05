package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/resources"
	"github.com/hashicorp/go-hclog"
)

type Client struct {
	client *ccv3.Client
	logger hclog.Logger
}

func New(logger hclog.Logger) (*Client, error){
	envClient, err := getEnvClient()
	if err != nil {
		return nil, err
	}

	return &Client{
		client: envClient,
		logger: logger,
	}, nil
}

func (c *Client) listWarnings(warn ccv3.Warnings) {
	if len(warn) > 0 {
		for _, w := range warn {
			c.logger.Warn("Cloud Foundry warning", "warning", w)
		}
	}
}

func (c *Client) CloudControllerURL() interface{} {
	return c.client.CloudControllerURL
}

func (c *Client) GetDomainsByName(names ...string) (domains []resources.Domain, err error) {
	var warn ccv3.Warnings
	domains, warn, err = c.client.GetDomains(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: names,
	})
	c.listWarnings(warn)
	return domains, err
}

func (c *Client) CfClient() *ccv3.Client {
	return c.client
}

func (c *Client) GetProcessInstances(guid string) ([]ccv3.ProcessInstance, error) {
	processInstances, warn, err := c.client.GetProcessInstances(guid)
	c.listWarnings(warn)
	return processInstances, err
}