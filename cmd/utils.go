package main

import (
	"time"

	"github.com/rancher/wrangler/v3/pkg/genericcondition"
)

func getLatestWranglerObjectCondition(conditions []genericcondition.GenericCondition) *genericcondition.GenericCondition {
	if len(conditions) == 0 {
		return nil
	}

	var latestCondition *genericcondition.GenericCondition
	var latestTime time.Time

	// Parse the time and get the latest condition
	for _, condition := range conditions {
		conditionTime, err := time.Parse(time.RFC3339, condition.LastUpdateTime)
		if err != nil {
			continue
		}

		if latestCondition == nil || conditionTime.After(latestTime) {
			latestCondition = &condition
			latestTime = conditionTime
		}
	}

	return latestCondition
}
