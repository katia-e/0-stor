package benchers

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/benchmark/config"
	"github.com/zero-os/0-stor/client"
)

//WriteBencher represents a writing benchmarker
type WriteBencher struct {
	client     *client.Client
	scenario   *config.Scenario
	scenarioID string
	keys       [][]byte
	value      []byte
}

// NewWriteBencher returns a new WriteBencher
func NewWriteBencher(scenarioID string, scenario *config.Scenario) (Benchmarker, error) {
	wb := new(WriteBencher)

	err := scenario.Validate()
	if err != nil {
		log.Errorf("Error validating scenario: %v", err)
		return nil, fmt.Errorf("Scenario %s failed: %v", scenarioID, err)
	}
	wb.scenarioID = scenarioID
	wb.scenario = scenario
	var ops int
	if scenario.BenchConf.Operations <= 0 {
		ops = defaultOperations
	} else {
		ops = scenario.BenchConf.Operations
	}

	// generate data
	for i := 0; i < ops; i++ {
		wb.keys = append(wb.keys, generateData(scenario.BenchConf.KeySize))
	}
	wb.value = generateData(scenario.BenchConf.ValueSize)

	// initializing client
	config.SetupClientConfig(&scenario.ZstorConf)
	wb.client, err = client.NewClientFromConfig(scenario.ZstorConf, 1)
	if err != nil {
		log.Errorf("Error creating client: %v", err)
		return nil, fmt.Errorf("Failed creating client: %v", err)
	}

	return wb, nil
}

//RunBenchmark implements Method.RunBenchmark
func (wb *WriteBencher) RunBenchmark() (*Result, error) {
	if wb.client == nil {
		log.Error("zstor client is nil when trying to run a write bencher")
		return nil, fmt.Errorf("zstor client is nil")
	}

	var timeout <-chan time.Time
	if wb.scenario.BenchConf.Duration <= 0 {
		timeout = nil
	} else {
		timeout = time.After(time.Duration(wb.scenario.BenchConf.Duration) * time.Second)
	}

	// set up data aggregation interval
	interval, ok := ResultOptions[wb.scenario.BenchConf.Output]
	if !ok {
		interval = time.Second
	}

	var (
		tick            = time.Tick(interval * 1)
		start           time.Time
		counter         int64
		intervalCounter int64
		result          = &Result{}
		maxIteration    = len(wb.keys)
	)

	start = time.Now()
	for i := 0; ; i++ {
		// loop over the available keys
		key := wb.keys[i%maxIteration]

		select {
		case <-timeout:
			//timeout reached, make exit condition true
			timeout = nil
			i = maxIteration
		case <-tick:
			result.PerInterval = append(result.PerInterval, intervalCounter)
			intervalCounter = 0
		default:
			_, err := wb.client.Write(key, bytes.NewReader(wb.value))
			if err != nil {
				log.Errorf("Error write request to client: %v", err)
				return nil, err
			}
			intervalCounter++
			counter++
		}

		if timeout == nil && i >= maxIteration-1 {
			break
		}
	}
	result.Duration = Duration{time.Since(start)}
	result.Count = counter
	if intervalCounter != 0 {
		result.PerInterval = append(result.PerInterval, intervalCounter)
	}

	return result, nil
}
