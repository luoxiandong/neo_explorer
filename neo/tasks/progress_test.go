package tasks

import (
	"fmt"
	"math/big"
	"testing"
	"time"
)

func TestGetEstimatedRemainingTime(t *testing.T) {
	now := time.Now()
	maxIndex := int64(840)
	highestIndex := int64(5511583)
	m, _ := time.ParseDuration("-2s")

	percentage := new(big.Float).Quo(new(big.Float).SetInt64(maxIndex-30), new(big.Float).SetInt64(highestIndex))
	percentage = new(big.Float).Mul(percentage, big.NewFloat(100))

	bProgress    = Progress{
		InitTime: now.Add(m),
		InitPercentage: percentage,
	}

	GetEstimatedRemainingTime(maxIndex, highestIndex, &bProgress)
	fmt.Printf("%sBlock storage progress: %d/%d, %.4f%%\n",
		bProgress.RemainingTimeStr,
		maxIndex,
		highestIndex,
		bProgress.Percentage,
	)
}