// Package cost turns a resource recommendation into a dollar figure:
// how much a container's configured CPU/memory *request* costs beyond
// what its observed usage recommends. Waste is scored on requests, not
// limits, because requests are what a scheduler reserves capacity for
// (and what cluster autoscalers size nodes against) — an over-set limit
// risks nothing but a slow OOM kill, an over-set request burns real
// node capacity every hour it sits idle.
package cost

const hoursPerMonth = 730 // 365.25 * 24 / 12

// Rates is $ per unit-hour. Defaults approximate blended general-purpose
// on-demand VM pricing — illustrative, not a specific cloud's price
// list. Override via config to match actual billing.
type Rates struct {
	CPUCorePerHour float64
	MemGiBPerHour  float64
}

func DefaultRates() Rates {
	return Rates{CPUCorePerHour: 0.033, MemGiBPerHour: 0.0045}
}

// MonthlyWaste estimates $/month spent on requested CPU/memory beyond
// the recommended amount. Under-provisioning (configured < recommended)
// isn't waste — it's a reliability risk, and this project only scores
// the cost side, not the risk side.
func MonthlyWaste(configuredCPUMilli, recommendedCPUMilli, configuredMemBytes, recommendedMemBytes int64, rates Rates) float64 {
	var waste float64
	if configuredCPUMilli > recommendedCPUMilli {
		cores := float64(configuredCPUMilli-recommendedCPUMilli) / 1000
		waste += cores * rates.CPUCorePerHour * hoursPerMonth
	}
	if configuredMemBytes > recommendedMemBytes {
		gib := float64(configuredMemBytes-recommendedMemBytes) / (1 << 30)
		waste += gib * rates.MemGiBPerHour * hoursPerMonth
	}
	return waste
}
