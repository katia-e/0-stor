"""
    Package config includes functions to set up configuration for benchmarking scenarios
"""
import pdb
import sys
import time
import os
from re import split
from copy import deepcopy
import yaml
from lib.zstor_local_setup import SetupZstor
from subprocess import run
from subprocess import check_output

# list of supported benchmark parameters
Parameters = {'block_size', 
                'key_size', 
                'value_size', 
                'clients', 
                'mode',         # compression 
                'method',
                'block_size',
                'data_shards',
                'parity_shards'}

Profiles = { 'cpu', 'mem', 'trace', 'block'}

Default_data_start_port = '1200'
Default_meta_start_port = '1300'
class Config:
    """
    Class Config includes functions to set up environment for the benchmarking:
        - deploy zstor servers
        - config zstor client
        - iterate over range of benchmark parameters

    @template contains zerostor client config
    @benchmark defines iterator over provided benchmarks
    """
    def __init__(self, config_file):
        # read config yaml file
        with open(config_file, 'r') as stream:
            try:
                config = yaml.load(stream)
            except yaml.YAMLError as exc:
                sys.exit(exc)

        # fetch template config for benchmarking
        self._template0 = config.pop('template', None)
        self.restore_template()        

        if not self.template:
            sys.exit('no template config given')

        # extract benchmarking parameters
        self.benchmark = iter(self.benchmark_generator(config.pop('benchmarks', None)))
        
        # extract profiling parameter
        self.profile = config.pop('profile', None)

        if self.profile and (self.profile not in Profiles):
            sys.exit("orchestrator config: profile mode '%s' is not supported"%self.profile)
        
        self.count_profile = 0

        self.deploy = SetupZstor()
    
    def new_profile_dir(self,path=""):
        """
        Creates new directory for profile information in given path and dumps current config
        """
        if self.profile:
            directory = '%s/profile_information'%path
            if not os.path.exists(directory):
                os.makedirs(directory)
            directory = '%s/profile_%s'%(directory,str(self.count_profile))         
            if not os.path.exists(directory):
                os.makedirs(directory)
            file = "%s/config.yaml"%directory
            with open(file, 'w+') as outfile:
                yaml.dump({'scenarios': {'scenario': self.template}}, 
                            outfile, 
                            default_flow_style=False, 
                            default_style='')             
            self.count_profile += 1    
            return directory
        return "" 

    def benchmark_generator(self,benchmarks):
        """
        Iterate over list of benchmarks
        """     
        if benchmarks:
            for bench in benchmarks:
                yield BenchmarkPair(bench)        
        else:
            yield BenchmarkPair()
        
        
    def alter_template(self, id, val):        
        """
        Recurcively search and ppdate @id config field with new value @val
        """
        def replace(d, id, val):
            for key in list(d.keys()):
                v = d[key]
                #import ipdb; ipdb.set_trace()
                if isinstance(v, dict):
                    if replace(v, id, val):
                        return True
                else:
                    if key == id:
                        parameter_type = type(d[key])
                        try:
                            d[key] = parameter_type(val)
                        except ValueError:
                            sys.exit("orchestrator config: cannot convert val = {} to type {}".format(val,parameter_type))
                        return True
            return False
        if not replace(self.template, id, val):
            sys.exit("orchestrator config: parameter %s is not supported"%id)

    def restore_template(self):
        self.template = deepcopy(self._template0)


    def save(self, file_name):
        """
        Save current config to file
        """
        # prepare config for output
        output = {'scenarios': {'scenario': self.template}}

        # write scenarios to a yaml file 
        with open(file_name, 'w+') as outfile:
            yaml.dump(output, outfile, default_flow_style=False, default_style='')     


    def update_deployment_config(self):
        """ 
        Fetch current zstor server deployment config
        """

        try:
            self.datastor =  self.template['zstor_config']['datastor']
            distribution = self.datastor['pipeline']['distribution']
            self.data_shards_nr=distribution['data_shards'] + distribution['parity_shards']
        except:
            print("orchestrator config: distribution config is not given correctly")
            raise
        
        try:
            self.metastor  = self.template['zstor_config']['metastor']['db']
            self.meta_shards_nr = self.metastor['meta_shards_nr']
        except:
            print("orchestrator config: number of metastor servers is not given")
            raise

        self.data_start_port = self.get_port(self.datastor.pop('data_start_port', Default_data_start_port))
        self.meta_start_port = self.get_port(self.metastor.pop('meta_start_port', Default_meta_start_port))

    def deploy_zstor(self):
        self.deploy.run_zstordb_servers(servers=self.data_shards_nr,
                                    start_port=self.data_start_port,)
        self.deploy.run_etcd_servers(servers=self.meta_shards_nr,
                                    start_port=self.meta_start_port)
        self.datastor.update({'shards': self.deploy.data_shards})
        self.metastor.update({'endpoints': self.deploy.meta_shards})                                      

    def stop_zstor(self):
        self.deploy.stop_etcd_servers()
        self.deploy.stop_zstordb_servers()
        self.deploy.cleanup()

    @staticmethod
    def get_port(addr=None):
        if not addr:
            return
        try:
            return port2int(addr)
        except:
            return port2int(split(':', addr)[-1])        

    @staticmethod
    def fix_port_list(start_port, servers):
        """ 
        Correct list of addresses for the local server deployment.
        
        From the given port +1 for each next port.
        """
        host = "127.0.0.1"
      
        new_addrs=[]
        for port in range(start_port, start_port+servers):
            new_addrs.append("%s:%s"%(host,port))
        return new_addrs

    def wait_local_servers_to_start(self):
        """ 
        Check whether ztror and etcd servers are listening on the ports 
        """
        addrs = self.deploy.data_shards + self.deploy.meta_shards
        servers = 0                
        timeout = time.time() + 20
        while servers<len(addrs):
            servers = 0
            for addr in addrs:
                port = ':%s'%split(':',addr)[-1]
                try:
                    responce = check_output(['lsof', '-i', port])
                except:
                    responce=0
                if responce:
                    servers+=1
                if time.time() > timeout:
                    print("timeout error: couldn't run all required servers")
                    break   
        
class Benchmark():
    """ 
    Benchmark class is used defines and validates benchmark parameter   
    """
    def __init__(self, parameter={}):
        if parameter:
            try:
                self.id = parameter['id']              
            except:
                print("parameter", parameter)
                sys.exit("invalid benchmark: parameter id field is missing")
            if not self.id:
                sys.exit("Invalid benchmark: parameter id is empty")             
            if self.id not in Parameters:
                sys.exit("invalid benchmark: {0} is not supported".format(self.id))
            try:
                self.range = split("\W+", parameter['range'])
            except:
                sys.exit("invalid benchmark: parameter range field is missing")
            if not range:
                sys.exit("invalid benchmark: no range is given for {0}".format(self.id))
        else:
            # return empty Benchmark
            self.range = [' ']
            self.id = ''

    def empty(self):
        if (len(self.range) == 1) and not self.id:
            return True
        return False

class BenchmarkPair():
    """
    BenchmarkPair defines prime and secondary parameter for benchmarking
    """
    def __init__(self, bench_pair={}):
        if bench_pair:
            # extract parameters from a dictionary
            self.prime = Benchmark(bench_pair.pop('prime_parameter', None))
            self.second = Benchmark(bench_pair.pop('second_parameter', None))

            if not self.prime.empty and self.prime.id == self.second.id:
                sys.exit("error: primary and secondary parameters should be different")
            
            if self.prime.empty() and not self.second.empty():
                sys.exit("error: if secondary parameter is given, primary parameter has to be given")
        else:
            # define empty benchmark
            self.prime = Benchmark()
            self.second = Benchmark()                                               

def port2int(port):
    try:
        port = int(port)
    except ValueError:
        sys.exit("error config: wrong port format")        
    return port             