// Package scaler decides target replica counts from observed CPU
// utilization, the same ratio-based formula the built-in
// HorizontalPodAutoscaler uses: desired = ceil(current * observed/target).
package scaler

import "math"

// Decide returns the replica count current should move to, given
// currentUtilPercent (0 means no usage data yet — hold steady) against
// targetUtilPercent, clamped to [min, max]. tolerance (e.g. 0.1 for 10%)
// suppresses churn from ratios that are close enough to 1 to be noise.
func Decide(current int32, currentUtilPercent, targetUtilPercent, tolerance float64, min, max int32) int32 {
	if currentUtilPercent <= 0 || targetUtilPercent <= 0 {
		return current
	}
	ratio := currentUtilPercent / targetUtilPercent
	if math.Abs(ratio-1) <= tolerance {
		return current
	}
	desired := int32(math.Ceil(float64(current) * ratio))
	if desired < min {
		desired = min
	}
	if desired > max {
		desired = max
	}
	return desired
}
