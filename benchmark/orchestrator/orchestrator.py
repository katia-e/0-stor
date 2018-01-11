"""
    Orchestrator controls the running of benchmarking process,
    aggregating results and producing report.
"""

import sys
from argparse import ArgumentParser
from argparse import SUPPRESS
from yaml import dump
from subprocess import run
from lib import Config
from lib import Report
from lib.bench_client import Bench_client
import time

def main(argv):    
    parser = ArgumentParser(epilog="""
        Orchestrator controls the benchmarking process,
        aggregating results and producing report.
    """, add_help=False)
    parser.add_argument('-h', '--help', action='help',
                    help='help for orchestrator')    
    parser.add_argument('-C','--conf', 
                        metavar='string',
                        default='orchConfig.yaml',
                        help='path to the config file (default orchConfig.yaml)')
    parser.add_argument('--out', 
                        metavar='string',
                        default='report',
                        help='directory where the benchmark report will be written (default ./report)')

    args = parser.parse_args()
    input_config = args.conf
    report_directory = args.out
   
    # path where config for scenarios is written
    output_config = "scenariosConf.yaml"
    # path to the benchmark results
    result_benchmark_file = "benchmarkResult.yaml"          

    print('********************')
    print('****Benchmarking****')
    print('********************')

    # extract config information
    config = Config(input_config)

    # initialise report opject
    report = Report(report_directory)

    # initialize benchmarking tool
    benchmarking = Bench_client()

    # loop over all given benchmarks
    try:
        while True:
            # switch to the next benchmark config
            benchmark = next(config.benchmark)

            # define a new data collection        
            report.init_aggregator(benchmark)

            # loop over range of the secondary parameter
            for val_second in benchmark.second.range:
                report.aggregator.new()

                # alter the template config if secondary parameter is given
                if not benchmark.second.empty():
                    config.alter_template(benchmark.second.id, val_second)

                # loop over the prime parameter
                for val_prime in benchmark.prime.range:    
                    # alter the template config if prime parameter is given
                    if not benchmark.prime.empty():
                        config.alter_template(benchmark.prime.id, val_prime)  
                    
                    
                    # update deployment config 
                    config.update_deployment_config()

                    # deploy zstor
                    config.deploy_zstor()

                    # update config file
                    config.save(output_config)

                    # wait for servers to start
                    config.wait_local_servers_to_start()                                  
                    
                    # perform benchmarking 
                    benchmarking.run(config=output_config, 
                                     out=result_benchmark_file,
                                     profile=config.profile, 
                                     profile_dir=config.new_profile_dir(report_directory))                    
                    # stop zstor
                    config.stop_zstor()

                    # aggregate results
                    report.aggregate(result_benchmark_file)

                    # add timeplots to the report
                    report.add_timeplot()
            
            # add results of the benchmarking to the report
            report.add_aggregation()
            config.restore_template()
    except StopIteration:           # Note 4
        print("Benchmarking is done")

if __name__ == '__main__':
    main(sys.argv[1:])    