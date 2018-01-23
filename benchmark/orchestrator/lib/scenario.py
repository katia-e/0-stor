
import yaml

TIME_UNITS = {'per_second': 1,
              'per_minute': 60,
              'per_hour': 3600}

class InvalidBenchmarkResult(Exception):
    """ Defines invalid benchmark exception"""
    pass

class Scenario:
    """ 
    Scenario intcudes tools to fetch and 
        validate output of the benchmark 
    """

    def __init__(self, input_file=''):
        self.results = []
        self.scenario = {}
        self.config = {}
        if input_file:
            self.load_result(input_file)

    def load_result(self, input_file):
        """ Load output from yaml file to dictionary @self.scenario """\

        with open(input_file, 'r') as stream:
            try:
                scenarios = yaml.load(stream)['scenarios']
            except yaml.YAMLError as exc:
                raise exc
        keys = [k for k in scenarios]
        if len(keys) != 1:
            raise InvalidBenchmarkResult('output for exactly one scenario is expected in this context')
        self.scenario = scenarios[keys[0]]
        err = self.scenario.get('error', None)
        if err:
            raise InvalidBenchmarkResult("benchmark exited with error: %s"%err)
        self._validate()

    def _validate(self):
        """ Validate fields of the output file """
        
        self.config = self.scenario.get('scenario', None)
        if not self.config:
            raise InvalidBenchmarkResult("scenario config is missing")

        self.zstor_config = self.config.get('zstor_config', None)
        if not self.zstor_config:
            raise InvalidBenchmarkResult("zstor_config is missing")

        self.bench_config = self.config.get('bench_config', None)     
        if not self.bench_config:
            raise InvalidBenchmarkResult("bench_config is missing")

        self.results = self.scenario.get('results', None)
        if not self.results:
            raise InvalidBenchmarkResult("results are missing")

        self.result_output = self.bench_config.get('result_output', None)

        if not self.result_output or (self.result_output not in TIME_UNITS):
            for result in self.results:
                result['perinterval'] = []

def filter_dict(dictionary, filter_keys):
    """
    Recursively delete keys from dictionary.
    @ filter_keys specifies list  of keys
    """

    def filter_dict(dictionary, filter_keys):
        """ Delete @filter_keys from @dictionary """

        for key in list(dictionary.keys()):
            val = dictionary[key]
            if key in filter_keys:
                dictionary.pop(key, None)
            else:
                if isinstance(val, dict):
                    filter_dict(val, filter_keys)

    filter_dict(dictionary, filter_keys)