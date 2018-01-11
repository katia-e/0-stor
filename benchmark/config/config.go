//Package config defines a packeg used to set up client config.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
	validator "gopkg.in/validator.v2"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	// ErrZeroDataShards represents an error where no data shards were found in config
	ErrZeroDataShards = errors.New("No data shards in config")

	// ErrZeroMetaShards represents an error where no data shards were found in config
	ErrZeroMetaShards = errors.New("No meta shards in config")

	// ErrNotEnoughDistributionShards represents an error where there were not enough
	// data shards for the specified distribution data + parity shards
	ErrNotEnoughDistributionShards = errors.New("Not enough data shards for Pipeline Distribution config")
)

// ClientConf represents a client banchmark config
type ClientConf struct {
	Scenarios map[string]Scenario `yaml:"scenarios" validate:"nonzero"`
}

// Validate validates a ClientConf
func (clientConf *ClientConf) Validate() error {
	if len(clientConf.Scenarios) == 0 {
		return errors.New("Client config is empty")
	}

	return nil
}

// Scenario represents a scenario
type Scenario struct {
	ZstorConf client.Config   `yaml:"zstor_config" validate:"nonzero"`
	BenchConf BenchmarkConfig `yaml:"bench_config" validate:"nonzero"`
}

// Validate validates a scenario
func (sc *Scenario) Validate() error {
	return sc.BenchConf.Validate()
}

// SetupClientConfig sets up the client.Client for a benchmark.
// Removes IYO fields. (Benchmarks uses no-auth zstordb's)
// Sets random namespace if empty
func SetupClientConfig(c *client.Config) {
	// empty IYO fields
	c.IYO = itsyouonline.Config{}

	// set namespace if not provided
	if c.Namespace == "" {
		c.Namespace = "b-" + randomSuffix(4)
	}
}

func randomSuffix(n int) string {
	var s string
	for i := 0; i < n; i++ {
		s = s + strconv.Itoa(rand.Intn(10))
	}

	return s
}

// BenchmarkConfig represents benchmark configuration
type BenchmarkConfig struct {
	Method     string `yaml:"method" validate:"nonzero"`
	Output     string `yaml:"result_output"`
	Duration   int    `yaml:"duration"`
	Operations int    `yaml:"operations"`
	Clients    int    `yaml:"clients"`
	KeySize    int    `yaml:"key_size" validate:"nonzero"`
	ValueSize  int    `yaml:"value_size" validate:"nonzero"`
}

// Validate validates a BenchmarkConfig
func (bc *BenchmarkConfig) Validate() error {
	if bc.Duration <= 0 && bc.Operations <= 0 {
		return fmt.Errorf("duration or operations was not provided")
	}

	return validator.Validate(bc)
}

// FromReader returns client config from a given reader
func FromReader(r io.Reader) (*ClientConf, error) {
	clientConf := &ClientConf{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// unmarshal
	if err := yaml.Unmarshal(b, clientConf); err != nil {
		return nil, err
	}

	// validate
	if err := clientConf.Validate(); err != nil {
		return nil, err
	}

	return clientConf, nil
}
