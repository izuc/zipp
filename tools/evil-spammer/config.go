package main

import (
	"time"

	"github.com/izuc/zipp/client/evilwallet"
)

// Nodes used during the test, use at least two nodes to be able to doublespend
var (
	// urls = []string{"http://bootstrap-01.feature.zipp.org:8080", "http://vanilla-01.feature.zipp.org:8080", "http://drng-01.feature.zipp.org:8080"}
	urls = []string{"http://localhost:8080", "http://localhost:8090"}
)

var (
	Script = "basic"

	customSpamParams = CustomSpamParams{
		ClientUrls:            urls,
		SpamTypes:             []string{"blk"},
		Rates:                 []int{1},
		Durations:             []time.Duration{time.Second * 20},
		BlkToBeSent:           []int{0},
		TimeUnit:              time.Second,
		DelayBetweenConflicts: 0,
		Scenario:              evilwallet.Scenario1(),
		DeepSpam:              false,
		EnableRateSetter:      true,
	}
	quickTest = QuickTestParams{
		ClientUrls:            urls,
		Rate:                  100,
		Duration:              time.Second * 30,
		TimeUnit:              time.Second,
		DelayBetweenConflicts: 0,
		EnableRateSetter:      true,
	}
)
