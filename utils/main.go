package utils

import "sort"

// Helper function to calculate the maximum gas used.
func CalculateMax(data []ContractGasInfo) uint64 {
	if len(data) == 0 {
		return 0
	}
	max := data[0].GasUsed
	for _, item := range data {
		if item.GasUsed > max {
			max = item.GasUsed
		}
	}
	return max
}

// Helper function to calculate the minimum gas used.
func CalculateMin(data []ContractGasInfo) uint64 {
	if len(data) == 0 {
		return 0
	}
	min := data[0].GasUsed
	for _, item := range data {
		if item.GasUsed < min {
			min = item.GasUsed
		}
	}
	return min
}

// Helper function to calculate the median gas used.
func CalculateMedian(data []ContractGasInfo) uint64 {
	if len(data) == 0 {
		return 0
	}
	// Sort the data
	sort.Slice(data, func(i, j int) bool {
		return data[i].GasUsed < data[j].GasUsed
	})
	// Find the middle value
	middle := len(data) / 2
	if len(data)%2 == 0 {
		// Average of two middle values if even number of elements
		return (data[middle-1].GasUsed + data[middle].GasUsed) / 2
	}
	return data[middle].GasUsed
}

// Helper function to calculate the average gas used.
func CalculateAverage(data []ContractGasInfo) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := uint64(0)
	for _, item := range data {
		sum += item.GasUsed
	}
	return float64(sum) / float64(len(data))
}
