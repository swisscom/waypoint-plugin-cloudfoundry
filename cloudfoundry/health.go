package cloudfoundry

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"fmt"
	proto "github.com/hashicorp/waypoint-plugin-sdk/proto/gen"
)

func (c *Client) GetHealthByGUID(appGuid string) (result *proto.StatusReport, err error){
	result = &proto.StatusReport{}
	processes, warn, err := c.client.GetApplicationProcesses(appGuid)
	if err != nil {
		return nil, fmt.Errorf("error getting application processes: %v", err)
	}

	c.listWarnings(warn)

	processInstancesCount := 0
	statusMap := map[constant.ProcessInstanceState]int{}

	for _, proc := range processes {
		// Get Health Check result
		pInstances, warn, err := c.client.GetProcessInstances(proc.GUID)
		if err != nil {
			return nil, fmt.Errorf("error getting process instance for %s (part of %s): %s",
				proc.GUID,
				appGuid,
				err,
			)
		}
		c.listWarnings(warn)

		for _, pi := range pInstances {
			statusMap[pi.State]++
			processInstancesCount++
		}
	}

	if statusMap[constant.ProcessInstanceRunning] == processInstancesCount {
		result.Health = proto.StatusReport_READY
		result.HealthMessage = "all processes are reporting ready"
	} else if statusMap[constant.ProcessInstanceCrashed] == processInstancesCount {
		result.Health = proto.StatusReport_DOWN
		result.HealthMessage = "all processes are crashed"
	} else if statusMap[constant.ProcessInstanceStarting] == processInstancesCount {
		result.Health = proto.StatusReport_ALIVE
		result.HealthMessage = "all processes are starting"
	} else if statusMap[constant.ProcessInstanceDown] == processInstancesCount {
		result.Health = proto.StatusReport_DOWN
		result.HealthMessage = "all processes are reporting down"
	} else {
		result.Health = proto.StatusReport_PARTIAL
		result.HealthMessage = fmt.Sprintf(
			"all processes are reporting mixed status: %v",
			statusMap,
		)
	}
	return
}
