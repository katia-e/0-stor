package benchers

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/benchmark/config"
	"github.com/zero-os/0-stor/client"
)

//ReadBencher represents a reading benchmarker
type ReadBencher struct {
	client     *client.Client
	scenario   *config.Scenario
	scenarioID string
	keys       [][]byte
	value      []byte
}

// NewReadBencher returns a new ReadBencher
func NewReadBencher(scenarioID string, scenario *config.Scenario) (Benchmarker, error) {
	rb := new(ReadBencher)

	// validate scenario config
	err := scenario.Validate()
	if err != nil {
		log.Errorf("Error validating scenario: %v", err)
		return nil, fmt.Errorf("Scenario %s failed: %v", scenarioID, err)
	}

	// set bencher fields
	rb.scenarioID = scenarioID
	rb.scenario = scenario
	var ops int
	if scenario.BenchConf.Operations <= 0 {
		ops = defaultOperations
	} else {
		ops = scenario.BenchConf.Operations
	}

	// generate data
	for i := 0; i < ops; i++ {
		rb.keys = append(rb.keys, generateData(scenario.BenchConf.KeySize))
	}
	rb.value = generateData(scenario.BenchConf.ValueSize)

	// initializing client
	config.SetupClientConfig(&scenario.ZstorConf)
	rb.client, err = client.NewClientFromConfig(scenario.ZstorConf, 1)
	if err != nil {
		log.Errorf("Error creating client: %v", err)
		return nil, fmt.Errorf("Failed creating client: %v", err)
	}

	// set testdata to client
	for _, key := range rb.keys {
		_, err := rb.client.Write(key, bytes.NewReader(rb.value))
		if err != nil {
			return nil, err
		}
	}

	return rb, nil
}

// RunBenchmark runs the read benchmarker
func (rb *ReadBencher) RunBenchmark() (*Result, error) {
	if rb.client == nil {
		log.Error("zstor client is nil when trying to run a read bencher")
		return nil, fmt.Errorf("zstor client is nil")
	}

	var timeout <-chan time.Time
	if rb.scenario.BenchConf.Duration <= 0 {
		timeout = nil
	} else {
		timeout = time.After(time.Duration(rb.scenario.BenchConf.Duration) * time.Second)
	}

	// set up data aggregation interval
	interval, ok := ResultOptions[rb.scenario.BenchConf.Output]
	if !ok {
		interval = time.Second
	}

	var (
		tick            = time.Tick(interval)
		start           time.Time
		counter         int64
		intervalCounter int64
		result          = &Result{}
		maxIteration    = len(rb.keys)
	)

	start = time.Now()
	for i := 0; ; i++ {
		// loop over the available keys
		key := rb.keys[i%maxIteration]

		select {
		case <-timeout:
			//timeout reached, make exit condition true
			timeout = nil
			i = maxIteration
		case <-tick:
			result.PerInterval = append(result.PerInterval, intervalCounter)
			intervalCounter = 0
		default:
			buf := bytes.NewBuffer(nil)
			err := rb.client.Read(key, buf)
			if err != nil {
				log.Errorf("Error read request to client: %v", err)
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
