package cmd

import (
	"errors"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/benchmark/benchers"
	"github.com/zero-os/0-stor/benchmark/config"
)

//BenchmarkFlags defines flags
var BenchmarkFlags struct {
	confFile         string
	benchmarkOutPath string
	profileOutPath   string
	profileMode      string
}
var (
	// RootCmd creates flags
	RootCmd = &cobra.Command{
		Use:   "performance testing",
		Short: "runs benchmarking and profiling of a zstor client",
		Long: `

		Profiling and benchmarking of the zstor client is implemented.
		The result of benchmarking will be described in YAML format and written to file.

		Profiling mode is given using the --profile-mode flag, taking one of the following options:
			+ cpu
			+ mem
			+ trace
			+ block
		In case --profile-mode is not given, no profiling will be performed.

		Output directory for profiling is given by --out-profile flag.

		Config file used to initialize the benchmarking is given by --conf flag.
		Default config file is clientConf.yaml

		Output file for the benchmarking result can be given by --out-benchmark flag.
		Default output file is benchmark.yaml
	`,
		Run: func(cmd *cobra.Command, args []string) {
			root(cmd)
		},
	}
)

func init() {
	RootCmd.Flags().StringVarP(&BenchmarkFlags.confFile, "conf", "C", "config.yaml", "path to a config file")
	RootCmd.Flags().StringVar(&BenchmarkFlags.benchmarkOutPath, "out-benchmark", "benchmark.yaml", "path and filename where benchmarking results are written")
	RootCmd.Flags().StringVar(&BenchmarkFlags.profileOutPath, "out-profile", "profile", "path where profiling files are written")
	RootCmd.Flags().StringVar(&BenchmarkFlags.profileMode, "profile-mode", "", "enable profiling mode, one of [cpu, mem, trace, block]")
}

func root(cmd *cobra.Command) {
	// get configuration
	log.Info("Reading config...")
	clientConf, err := readConfig(BenchmarkFlags.confFile)
	if err != nil {
		log.Fatal(err)
	}

	// Start profiling if profiling flag is given
	switch BenchmarkFlags.profileMode {
	case "cpu":
		defer profile.Start(profile.ProfilePath(BenchmarkFlags.profileOutPath), profile.CPUProfile).Stop()
	case "mem":
		defer profile.Start(profile.ProfilePath(BenchmarkFlags.profileOutPath), profile.MemProfile).Stop()
	case "trace":
		defer profile.Start(profile.ProfilePath(BenchmarkFlags.profileOutPath), profile.TraceProfile).Stop()
	case "block":
		defer profile.Start(profile.ProfilePath(BenchmarkFlags.profileOutPath), profile.BlockProfile).Stop()
	default:
	}

	output := NewOutputFormat()

	//Run benchmarking for provided scenarios
	for scID, sc := range clientConf.Scenarios {
		log.Infof("Setting up benchmark `%s`...", scID)
		var b benchers.Benchmarker
		var err error
		var clients []benchers.Benchmarker
		var cc int // client count
		var wg sync.WaitGroup
		var results []*benchers.Result

		// define the type of bencher for the method given in scenario
		benchConstructor := benchers.GetBencherCtor(sc.BenchConf.Method)
		if benchConstructor == nil {
			err = errors.New("benchmark method not found")
			goto WriteResult
		}

		// get concurrent clients
		cc = sc.BenchConf.Clients
		if cc < 1 {
			cc = 1
		}
		clients = make([]benchers.Benchmarker, cc)
		results = make([]*benchers.Result, cc)

		// init clients
		for i := range clients {
			b, err = benchConstructor(scID, &sc)
			if err != nil {
				goto WriteResult
			}
			clients[i] = b
		}

		// run benchmarks concurrently
		log.Infof("Running benchmark `%s`...", scID)
		for i := range clients {
			wg.Add(1)
			go func(m benchers.Benchmarker, i int) {
				var result *benchers.Result
				result, err = b.RunBenchmark()
				results[i] = result
				wg.Done()
			}(clients[i], i)
		}
		wg.Wait()

		// collect results of the benchmarking cycle
	WriteResult:
		output.Scenarios[scID] = FormatOutput(results, sc, err)
	}

	// write results to file
	log.Info("Benchmarking done! Writing results!")
	err = writeOutput(BenchmarkFlags.benchmarkOutPath, output)
	if err != nil {
		log.Fatal(err)
	}
}

// readConfig reads a YAML config file and returns a config.ClientConf
// based on that file
func readConfig(path string) (*config.ClientConf, error) {
	yamlFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()

	// parse the config file to clientConf structure
	clientConf, err := config.FromReader(yamlFile)
	if err != nil {
		return nil, err
	}

	return clientConf, nil
}
