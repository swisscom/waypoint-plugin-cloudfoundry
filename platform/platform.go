package platform

import "github.com/hashicorp/waypoint-plugin-sdk/component"

var _ component.Platform = (*Platform)(nil)
var _ component.Generation = (*Platform)(nil)