package platform

import (
	"github.com/hashicorp/waypoint-plugin-sdk/component"
)

func (x *Deployment) URL() string {
	return x.Url
}

var _ component.DeploymentWithUrl = (*Deployment)(nil)
