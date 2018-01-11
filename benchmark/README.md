# zstor benchmark

`zstor benchmark` provides tools to analyse performance of `zstor`.

## Getting started

To start the benchmarking provide a [config file](#orchestrator-config-file) for `benchmark orchestrator` (`0-stor/benchmark/orchestrator/`) and run using optional parameters:
``` bash
python3 orchestrator --conf orchCong.yaml --out report
```

Here are the options of the orchestrator:
``` bash
optional arguments:
  -h, --help            help for orchestrator
  -C  --conf string     path to the config file (default rchConfig.yaml)
  --out string          directory where the benchmark report will be
                        written (default ./report)
```


## Components

## Benchmark orchestrator
  
To collect data for an informative benchmark report we need to repeatedly run `zstor` servers and `benchmark client` under multiple benchmarking scenarios and collect performance data. The benchmark scenarios, server and client config are provided in the orchestrator config.
  

### Orchestrator config file
Config file for the `orchestrator` consists of two parts:

  * `template` represents the template of the config file for the benchmark client.

  * `benchmarks` contains set of the benchmark parameter.
  Multiple benchmarks can be added to the final report. 

The config for each benchmark is marked by the `prime_parameter` and inside there can be an optional `second_parameter` defined. 
If no benchmark parameters are given, `template` will be directly used to run benchmarking and output the system throughput.
If only `prime_parameter` is given, `orchestrator` creates a plot of the system throughput versus values in `range` of the `prime parameter`.
If both `prime_parameter` and `second_parameter` are given, a few plots will be combined in the output figure, one for each value in `range` of `second_parameter`.

Also inside of the `prime_parameter` field the `id` specifies what zstor config field is being benchmarked.  
The `range` field specifies the different values for that zstor config field being used in the benchmarks.

Here is an example of the config file:
``` yaml
benchmarks:
- prime_parameter:
    id: value_size
    range: 128, 256, 512, 1024, 2048, 4096
  second_parameter:
    id: key_size
    range: 24, 48, 96    
- prime_parameter:
   id: value_size
   range: 128, 256, 512, 1024, 2048
  second_parameter:
    id: clients
    range: 1, 2, 3
template:
  zstor_config:
    datastor:
      data_start_port: 1200
      pipeline:
        block_size: 2048 
        compression:
          mode: default
        distribution:
          data_shards: 2
          parity_shards: 1
    metastor:
      db:
        meta_shards_nr: 2
        meta_start_port: 1300
      encryption:
        private_key: ab345678901234567890123456789012
  bench_config:
    clients: 1
    method: write
    result_output: per_second
    operations: 0
    duration: 3
    key_size: 48
    value_size: 128
profile: cpu
```
Port of the local host given in `data shards` is used by the orchestrator as a starting port for zstor servers deployment. Each next server uses the port +1.
Number of servers deployed is `distribution_data`+`distribution_parity`.

Port of the local host given in `meta shards` is used by the orchestrator as a starting port for etcd servers deployment. Each next server uses the port +1.

 
### Benchmark client

`Benchmark client` is written in `go` and contains ligic to collect benchmark information while creating load at the `zstor` server. Client config includes both `zstor` config and parameters of the benchmark. `Benchmark client` is called from `benchmark orchestrator`.

To use `zstore benchmark` independently run
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

Example of the config file is [here](https://github.com/zero-os/0-stor/blob/benchmark_orchestrator/benchmark/config/testconfigs/validConf.yaml).
