# zstor benchmark client

Benchmark client provides tools for benchmarking and profiling zstor client for various scenarios.

Configuration for benchmarking scenarios should be given in YAML format (see [example](#yaml-config-file) of the config file bellow).
Benchmarking program outputs results for all provided scenarios to a single output file in YAML format (see [example](#yaml-output-file) of the output file bellow). 


## Getting started

In order to start benchmarking program at least one [zstor server](https://github.com/zero-os/0-stor/blob/master/docs/gettingstarted.md) have to be set up.
When running a zstor server `--no-auth` flag has to be given for the sake of performance testing. This option allows to skip the authentification step. Client config still has to provide all necessary fields in order to pass the validation and successfully create a new zstor client.
```
zstordb --no-auth -D --listen 127.0.0.1:2379:12345 --data-dir stor1 --meta-dir stor1
```

Built benchmark client
```
go build
```

The following optional flags are defined
``` 
  -C  --conf string            path to a config file (default "clientConf.yaml")
  -h, --help                   help for this command
      --out-benchmark string   path and filename where benchmarking results are written (default "benchmark.yaml")
      --out-profile string     path where profiling files are written (default "profile")
      --profile-mode string    enable profiling mode, one of [cpu, mem, trace, block]

```

Start benchmarking with default input/output files
``` 
./client
```

Start benchmarking with optional input/output files
``` 
./client --conf "clientFonfig.yaml" --out-benchmark "dataset01.yaml"
```

Start benchmarking and profiling
``` 
./client --out-profile "outputProfileInfo" --profile-mode cpu
```

## YAML config file

Client config contains a list of scenarios. 
Each scenario is associated with a corresponding scenarioID and provides two sets of parameters: 
`zstor_config` and `bench_conf`.
Structure `zstor_config` are nessesary to create a `zstor client` and can be parsed into a type [client.Policy](https://github.com/zero-os/0-stor/blob/master/client/policy.go) of [zstor client package](https://github.com/zero-os/0-stor/tree/master/client). 


Structure `bench_conf` represents benchmarking specific configuration like duration of the performance test, number of operations, output format.

Key `method` provides the method for benchmarking and can take values
 + `read` - for reading from zstor
 + `write` - for writing to zstor

One of two parameters `duration` and `operations` has to be provided. If both are given, the benchmarking program terminates as soon as one of the following events occurs:
 + number of executed operations reached `operations`
 + timeout

 Key `result_output` specifies interval of the data collection and can take values
 + per_second
 + per_minute
 + per_hour

The following example of a config file represents two benchmarking scenarios `bench1` and `bench2`.


``` yaml
scenarios:
  bench1: # name of the first scenario
    zstor_config: # zstor config
      organization: "<IYO organization>"    #itsyou.online organization of the 0-stor
      namespace: <IYO namespace>            #itsyou.online namespace of the 0-stor
      iyo_app_id: "<an IYO app ID>"         #itsyou.online app/user id
      iyo_app_secret: "<an IYO app secret>" #itsyou.online app/user secret
      data_shards:
        - 127.0.0.1:12345
        - 127.0.0.1:12346
        - 127.0.0.1:12347
      meta_shards:
        - 127.0.0.1:2379
        - 127.0.0.1:22379
      block_size: 2048
      replication_nr: 2
      replication_max_size: 4096
      distribution_data: 2
      distribution_parity: 1
      compress: true
      encrypt: false
      encrypt_key: ab345678901234567890123456789012
    bench_conf:                    # config for benchmarking 
      method: write                # name of a benchmarking method
      result_output: per_second    # time interval of data collection
      duration: 10                 # duration of the benchmarking in seconds
      operations: 0                # number of operations
      key_size: 48                 # key size in bytes
      ValueSize: 128               # value size in bytes
  bench2: # name of the second scenario
    zstor_config: # zstor config
      organization: "<IYO organization>"
      namespace: <IYO namespace>
      iyo_app_id: "<an IYO app ID>"
      iyo_app_secret: "<an IYO app secret>"
      data_shards:
        - 127.0.0.1:12345
        - 127.0.0.1:12346
        - 127.0.0.1:12347
      meta_shards:
        - 127.0.0.1:12345
      block_size: 2048
      replication_nr: 2
      replication_max_size: 4096
      distribution_data: 2
      distribution_parity: 1
      compress: true
      encrypt: false
      encrypt_key: ab345678901234567890123456789012
    bench_conf:                    # config for benchmarking 
      method: write                # name of a benchmarking method
      result_output: per_minute    # time interval of data collection
      duration: 70                 # duration of the benchmarking in seconds
      operations: 20000            # number of operations
      key_size: 48                 # key size in bytes
      ValueSize: 128               # value size in bytes
```

## YAML output file

Benchmarking program writes results of the performance tests to an output file.
For each benchmarking scenario results are presented in the structure `result`, containing the following keys:
  + `count` - total number of operations executed
  + `duration` - total duration of the test
  + `perinterval` - number of iterations executed per time-unit

All scenario specific configuration is collected in `scenarioconf` key

``` yaml
bench1:
  result:
    count: 845
    duration: 10.004846224s
    perinterval:
    - 43
    - 14
    - 35
    - 107
    - 109
    - 108
    - 108
    - 109
    - 104
    - 107
    - 1
  scenarioconf:
    zstor_config:
      organization: ""
      namespace: <IYO namespace>
      iyo_app_id: ""
      iyo_app_secret: ""
      data_shards:
      - 127.0.0.1:12345
      - 127.0.0.1:12346
      - 127.0.0.1:12347
      meta_shards:
      - 127.0.0.1:12345
      block_size: 2048
      replication_nr: 2
      replication_max_size: 4096
      distribution_data: 2
      distribution_parity: 1
      compress: true
      encrypt: false
      encrypt_key: ab345678901234567890123456789012
    bench_conf:
      method: write
      result_output: per_minute
      duration: 70
      operations: 20000
      key_size: 48
      ValueSize: 128
  error: ""
bench2:
  result:
    count: 9746
    duration: 1m10.001785879s
    perinterval:
    - 8334
    - 1412
  scenarioconf:
    zstor_config:
      organization: ""
      namespace: <IYO namespace>
      iyo_app_id: ""
      iyo_app_secret: ""
      data_shards:
      - 127.0.0.1:12345
      - 127.0.0.1:12346
      - 127.0.0.1:12347
      meta_shards:
      - 127.0.0.1:12345
      block_size: 2048
      replication_nr: 2
      replication_max_size: 4096
      distribution_data: 2
      distribution_parity: 1
      compress: true
      encrypt: false
      encrypt_key: ab345678901234567890123456789012
    bench_conf:
      method: write
      result_output: per_minute
      duration: 70
      operations: 20000
      key_size: 48
      ValueSize: 128
  error: ""

```