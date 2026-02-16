package stats

import "time"

func MinResponseTime(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	min := durations[0]
	for _, d := range durations {
		if d < min {
			min = d
		}
	}

	return min
}
