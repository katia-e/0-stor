# zstor benchmark client

Benchmark client provides tools for benchmarking and profiling zstor client for various scenarios.

Configuration for benchmarking scenarios should be given in YAML format (see [example](#yaml-config-file) of the config file bellow).
Benchmarking program outputs results for all provided scenarios to a single output file in YAML format (see [example](#yaml-output-file) of the output file bellow). 


## Getting started
Install all necessary `zstor` components by running
```bash
make install
```
In order to start benchmarking program all necessary [zstor servers](https://github.com/zero-os/0-stor/blob/master/docs/gettingstarted.md) have to be set up.

Start the benchmarking
``` bash
zstorbench -C config.yaml --out-benchmark benchmark.yaml
```

`zstorbench` has the following options:
``` bash
  -C, --conf string            path to a config file (default "config.yaml")
  -h, --help                   help for performance
      --out-benchmark string   path and filename where benchmarking results are written (default "benchmark.yaml")
      --out-profile string     path where profiling files are written (default "profile")
      --profile-mode string    enable profiling mode, one of [cpu, mem, trace, block]
```


Start benchmarking with optional input/output files
``` bash
zstorbench --conf "input_config.yaml" --out-benchmark "output_benchmark.yaml"
```

Start benchmarking and profiling
``` bash
zstorbench --out-profile "outputProfileInfo" --profile-mode cpu
```

## Benchmark config

Client config contains a list of scenarios. 
Each scenario is associated with a corresponding scenarioID and provides two sets of parameters: 
`zstor_config` and `bench_conf`.
Structure `zstor_config` are nessesary to create a `zstor client` and can be parsed into a type [client.Policy](https://github.com/zero-os/0-stor/blob/master/client/policy.go) of [zstor client package](https://github.com/zero-os/0-stor/tree/master/client). 


Structure `bench_conf` represents such benchmark parameters like duration of the performance test, maximum number of operations, maximum benchmark duration and output format.
One of two parameters `duration` and `operations` has to be provided. If both are given, the benchmarking program terminates as soon as one of the following events occurs:
 + number of executed operations reached `operations`
 + timeout set by `duration` elapsed

`method` defines which operation is benchmarked:
 + `read` - for reading from zstor
 + `write` - for writing to zstor

`result_output` specifies interval of the data collection and can take values
 + per_second
 + per_minute
 + per_hour

The following example of a config file represents a benchmarking scenario `bench1`.

``` yaml
scenarios:
  bench1:
    zstor_config:
      namespace: adisk
      datastor:
        shards:
          - 127.0.0.1:1200
          - 127.0.0.1:1201
          - 127.0.0.1:1202
      metastor:
          shards:
            - 127.0.0.1:1300
            - 127.0.0.1:1301
          encryption:
            private_key: ab345678901234567890123456789012
      pipeline:
        block_size: 4096
        compression:
          mode: default
        encryption:
          private_key: ab345678901234567890123456789012
        distribution:
          data_shards: 2
          parity_shards: 1
    bench_config:
      method: write
      result_output: per_second
      duration: 5
      operations: 0
      key_size: 48
      value_size: 128
      clients: 1
```

## Output file

Benchmarking program writes results of the performance tests to an output file. All scenario configuration is collected in `scenario`. All numerical results can be fetched from `results`.

``` yaml
scenarios:
  bench1:
    results:
    - count: 367            # number of reads/writes performed during the benchmark
      duration: 5.0093327  # total duration of the benchmark
      perinterval:         # number of reads/writes per time unit
      - 84
      - 64
      - 67
      - 70
      - 82
    scenario:         # zstor config used for the benchmark
      zstor_config:
        iyo:
          organization: ""
          app_id: ""
          app_secret: ""
        namespace: adisk
        datastor:
          shards:
          - 127.0.0.1:45627
          - 127.0.0.1:49861
          - 127.0.0.1:37355
        metastor:
          shards:
          - 127.0.0.1:51583
          - 127.0.0.1:48843
        pipeline:
          block_size: 4096
          hashing:
            type: blake2b_256
            private_key: ""
          compression:
            mode: default
            type: snappy
          encryption:
            private_key: ab345678901234567890123456789012
            type: aes
          distribution:
            data_shards: 2
            parity_shards: 1
      bench_config:
        method: write
        result_output: per_second
        duration: 5
        operations: 0
        clients: 1
        key_size: 48
        value_size: 128
```