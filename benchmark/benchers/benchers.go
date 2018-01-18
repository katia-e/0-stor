/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package benchers provides methods to run benchmarking
package benchers

import (
	"math/rand"
	"strings"
	"time"

	"github.com/zero-os/0-stor/benchmark/config"
)

func init() {
	// seed random generator
	rand.Seed(time.Now().UnixNano())
}

const (
	// defaultKeyCount is used when BenchConf.Operations was not provided/invalid
	// when setting up the keys for the benchmark
	defaultKeyCount = 1000
)

var (
	// Methods represent name mapping for benchmarking methods
	benchers = map[string]BencherCtor{
		"read":  NewReadBencher,
		"write": NewWriteBencher,
	}
	// ResultOptions represent name mapping for benchmarking methods
	ResultOptions = map[string]time.Duration{
		"per_second": time.Second,
		"per_minute": time.Minute,
		"per_hour":   time.Hour,
	}
)

// BencherCtor represents a benchmarker constructor
type BencherCtor func(scenarioID string, conf *config.Scenario) (Benchmarker, error)

// Benchmarker represents benchmarking methods
type Benchmarker interface {
	// RunBenchmark starts the benchmarking
	RunBenchmark() (*Result, error)
}

// Result represents a benchmark result
type Result struct {
	Count       int64
	Duration    Duration
	PerInterval []int64
}

// Duration represents a duration of a test result
// used for custom YAML output
type Duration struct{ time.Duration }

// MarshalYAML implements yaml.Marshaler.MarshalYAML
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Seconds(), nil
}

// GetBencherCtor returns a BencherCtor that belongs to the provided method string
// if benchmarking method was not found, nil is returned
func GetBencherCtor(benchMethod string) BencherCtor {
	benchConstructor, ok := benchers[strings.ToLower(benchMethod)]
	if !ok {
		return nil
	}
	return benchConstructor
}

// generateData generates a byteslice of provided length
// filled with random data
func generateData(len int) []byte {
	data := make([]byte, len)
	rand.Read(data)
	return data
}
