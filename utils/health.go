package utils

import (
	proto "github.com/hashicorp/waypoint-plugin-sdk/proto/gen"
	"strings"
)

func HealthSummary(statusReport ...*proto.StatusReport) *proto.StatusReport {
	if statusReport == nil {
		return nil
	}

	result := proto.StatusReport{}
	result.Health = proto.StatusReport_READY
	var outMessage []string

	for _, r := range statusReport {
		outMessage = append(outMessage, r.HealthMessage)
		if isWorse(r.Health, result.Health){
			result.Health = r.Health
		}
	}

	result.HealthMessage = strings.Join(outMessage, ", ")
	result.External = statusReport[0].External
	return &result
}

func isWorse(health1 proto.StatusReport_Health, health2 proto.StatusReport_Health) bool {
	goodToWorse := []proto.StatusReport_Health{
		proto.StatusReport_READY, proto.StatusReport_ALIVE,
		proto.StatusReport_PARTIAL, proto.StatusReport_DOWN,
		proto.StatusReport_UNKNOWN,
	}

	var indices = make([]int, 2)

	for i, v := range goodToWorse {
		if v == health1 {
			indices[0] = i
		}
		if v == health2 {
			indices[1] = i
		}
	}

	return indices[0] >= indices[1]
}
