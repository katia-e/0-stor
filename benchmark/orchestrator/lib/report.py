"""
    Package report contains tools for collecting results of the benchmarking 
    and generating final report. 

    Output files show benchmarking results by means of tables and figures 
    allowing for visual representation. Report for each benchmark is added 
    to the output files as soon as the benchmark is finished.

    Two types of output files are avaliable: main report file and a timeplot
    collection. Main report file consists of performance measures and allows 
    for performance comparision among various scenarios.

    The performance metric, reflected in the report is throughput, 
    defined as an average data rate observed throughout the benchmark.

    Timeplot collection report soncists the scope of timeplots
    collected during the benchmarking. Timeplots show number of 
    operations per time unit, observes during the benchmark.
"""
import os
import matplotlib.pyplot as plt
import yaml
import sys

TimeUnits = {'per_second': 1, 
             'per_minute': 60, 
               'per_hour': 3600}

FilterKeys={'organization', 
            'namespace', 
            'iyo',  
            'shards', 
            'db', 
            'hashing'}

class Aggregator:
    """
    Aggregator aggregates average throughput over a set of benchmarks
    """
    def __init__(self, benchmark=None):
        self.benchmark = benchmark
        self.throughput= []

    def new(self):
        self.throughput.append([])

class Report:
    """
    Class Report is used to collect results of benchmarking
        and to create final report.
    """
    def __init__(self, directory='report', report='report.md', timeplots='timeplots.md'):
        self.directory = directory # set output directory for report files
        self.main_file =  "{0}/{1}".format(self.directory,report)
        self.timeplots_collection =  "{0}/{1}".format(self.directory, timeplots)

        if not os.path.exists(self.directory):
            os.makedirs(self.directory)  

        with open(self.main_file, 'w+') as outfile:
            outfile.write("# Benchmark report\n")     
            outfile.write("[Timeplot collection is here]({0})\n".format(timeplots))    

        with open(self.timeplots_collection, 'w+') as outfile:
            outfile.write("# Timeplot collection report\n")  
            outfile.write("[Main report in here]({0}) \n\n".format(report)) 

        self.reports_added = 0   # keep track of added reports
        self.timeplots_added = 0 # keep track of number of timeplots added 

    def init_aggregator(self, benchmark=None):
        self.aggregator = Aggregator(benchmark)

    def get_scenario_config(self, input_file):
        """
        Fetch benchmark scenario config
        """
        with open(input_file, 'r') as stream:
            try:
                scenarios = yaml.load(stream)['scenarios']
            except yaml.YAMLError as exc:
                sys.exit(exc)
        keys = [k for k in scenarios]
        if len(keys) != 1:
            sys.exit('result: exactly one scenario is expected in this context')
          
        self.scenario = scenarios[keys[0]]

        err = self.scenario.pop('error', None)
        if err:
            sys.exit("last benchmark exited with error: %s"%err )
        
        if len(scenarios) > 1:
            sys.exit("orchestrator config: expected only one scenario each time" )
        self.filter(self.scenario, FilterKeys)
        
    @staticmethod
    def filter(d, filter_keys):
        """
        Recurcively delete keys from dictionary.
        @ filter_keys specifies list  of keys
        """        
        def filter(d, filter_keys):
            for key in list(d.keys()):
                v = d[key]
                if key in filter_keys:
                    d.pop(key, None)    
                else:            
                    if isinstance(v, dict):
                        filter(v, filter_keys)

        filter(d, filter_keys) 


    def aggregate(self, input_file):
        self.get_scenario_config(input_file)
        th = self._get_throughput()      

        if self.aggregator.throughput: 
            self.aggregator.throughput[-1].append(th)
        else:
            self.aggregator.throughput.append(th)

    def _get_scenario_results(self):
        scenario = self.scenario.get('scenario')
        if not scenario:
            sys.exit("results: no scenario config")  

        zstor_config = scenario.get('zstor_config')
        if not zstor_config:
            sys.exit("results: no zstor_config")
        
        bench_config = scenario.get('bench_config')
        if not bench_config:
            sys.exit("results: no bench_config")  
        
        result_output = bench_config.get('result_output')
        if not bench_config:
            sys.exit("results: interval is not given (result_output)")          

        results = self.scenario.get('results')
        if not results:
            sys.exit("results: no results are provided")
        return scenario, results
        

    def _get_throughput(self):
        throughput = 0

        scenario, results = self._get_scenario_results()

        if not results:
            sys.exit("no results are provided")
        if not scenario:
            sys.exit("no scenario config")

        for result in results:    
            # get duration of the benchmarking
            try:
                duration = float(result['duration'])
            except:
                sys.exit('result:duration is not given, or format is not float')   
            if duration == 0:
                sys.exit("result: duration can't be 0")

            # number of operations in the benchmarking
            try:
                count = int(result['count'])
            except:
                exit('count is not given, or format is not int')   

            # get size of each value
            #import ipdb; ipdb.set_trace()
            try:
                value_size = int(scenario['bench_config']['value_size'])
            except:
                exit('orchestrator config: value size is not given, or format is not int')  
                    
            throughput += count*value_size/duration/len(results)

        return int(throughput)

    def add_aggregation(self):
        # count reports added to a report file
        self.reports_added += 1

        fig_name = 'fig' +str(self.reports_added) + '.png'

        # filter results form scenario config before dumping to the report
        self.filter(self.scenario, ['results'])
        
        with open(self.main_file, 'a+') as outfile:
            # refer the figure in the report
            outfile.write("\n # Report {0} \n".format(str(self.reports_added)))
            
            # add benchmark config
            outfile.write('**Benchmark config:** \n')
            outfile.write('```yaml \n')
            yaml.dump(self.scenario, outfile,default_flow_style=False)
            outfile.write('\n```')

        # check if more then one output was collected   
        if sum(map(len, self.aggregator.throughput)) > 1:
            # create a bar plot
            self._bar_plot( fig_name)

            # incerst bar plot to the report
            with open(self.main_file, 'a+') as outfile:
                outfile.write("\n![Fig: throughput vs parameter]({0})".format(fig_name))            

        # add the table of the data sets
        self._add_table()


    def _bar_plot(self, fig_name):
        # define range  from prime parameter
        ticks_labels = self.aggregator.benchmark.prime.range

        # af first results are plot vs counting number of samples
        rng = [i for i, tmp in enumerate(ticks_labels)]

        # number of data sets combined in the figure
        if len(self.aggregator.throughput) == 0:
            sys.exit("no results are included")

        max_throughput = max(max(self.aggregator.throughput))

        """ figure settings """       
        n_plots = len(self.aggregator.throughput[0]) # number of plots in the figure

        n_samples = len(rng) # number of samples for each data set
  
        width = rng[-1]/(n_samples*n_plots+1) # bar width
       
        gap = width/10  # gap between bars

        diff_y = 0.06 # minimal relative difference in throughput between neighboring bars
        label_y_gap = max_throughput/100

        # create figure
        fig, ax = plt.subplots()

        # limmit number of ticks to the number of samples
        plt.xticks(rng)

        # substitute tick labels
        ax.set_xticklabels(ticks_labels)
        
        # define color cycle 
        ax.set_color_cycle(['blue', 'red', 'green', 'yellow', 'black', 'brown'])

        ax.set_xlabel(self.aggregator.benchmark.prime.id)
        ax.set_ylabel('throughput, byte/s')

        # loop over data sets
        for i, th in enumerate(self.aggregator.throughput):
            # define plot label
            lb = " "
            if self.aggregator.benchmark.second.id:
                lb = "{0}={1}".format(self.aggregator.benchmark.second.id,
                                    self.aggregator.benchmark.second.range[i])

            # add bar plot to the figure
            ax.bar(rng, th, width, label=lb)
            lgd = ax.legend(loc='upper left', bbox_to_anchor=(1,1))
            #plt.tight_layout(pad=20)

            # add labels to bars
            for j, v in enumerate(th):
                if i:
                    if abs(v-self.aggregator.throughput[i-1][j])/max_throughput<diff_y:
                        continue
                ax.text(rng[j]-width/2, v+label_y_gap , str(v), color='blue', fontweight='bold')

            # shift bars for the next plot
            rng = [x+gap+width for x in rng]

        # label axes
        plt.savefig(self.directory+"/"+fig_name, bbox_extra_artists=(lgd,), bbox_inches='tight')       
        plt.close()

    def _add_table(self):         
        # add hidden table with data
        with open(self.main_file, 'a+') as outfile:
            # create a table
            outfile.write("""\n <h3> Throughput, byte/s: </h3>
            <head> 
                <style>
                    table, th, td {
                        border: 1px solid black;
                        border-collapse: collapse;
                    }
                    th, td {
                        text-align: left;    
                    }
                </style>
            </head>
            <table>  
                <tr> <th> """ + self.aggregator.benchmark.prime.id + "</th>")
            # add titles to the columns    
            for item in self.aggregator.benchmark.second.range:                
                if self.aggregator.benchmark.second.id:
                    outfile.write("<th> {0} = {1} </th>".format(self.aggregator.benchmark.second.id,item))
                else:
                    outfile.write("<th>  </th>")

                
            outfile.write(" </tr> ")

            # fill in the table
            for row, val in enumerate(self.aggregator.benchmark.prime.range):
                outfile.write("<tr> <th> {0} </th>".format(val))
                for col, tmp in enumerate(self.aggregator.benchmark.second.range):
                    outfile.write("<th> {0} </th>".format(str(self.aggregator.throughput[col][row])))
                outfile.write("</tr>")                    
                             


            outfile.write("\n </table></details>\n")


    def add_timeplot(self):
        """
        Add timeplots to the report
        """
        files = self._plot_per_interval()

        if len(files)>0:
            with open(self.timeplots_collection, 'a+') as outfile:
                outfile.write('\n**Config:**\n```yaml \n')
                yaml.dump(self.scenario, outfile, default_flow_style=False)
                outfile.write('\n```')
                outfile.write("\n _____________ \n".format(str(self.reports_added)))
                for file in files:
                    outfile.write("\n![Fig](../{0}) \n".format(file))

    def _plot_per_interval(self): 
        """
        Create timeplots
        """
        # file_names returns list of the output files
        file_names = []

        scenario, results = self._get_scenario_results()

        # time_unit_literal represents the time unit for aggregation of the results
        time_unit_literal = scenario['bench_config']['result_output']
        timeUnit = TimeUnits.get(time_unit_literal)
        
        if not timeUnit:
            sys.exit('results: result_output value is not supported')        
        
        for result in results:
            # duration of the benchmarking
            try:
                duration = float(result['duration'])
            except:
                exit('duration format is not valid')        

            # per_interval represents number of opperations per time unit
            per_interval = result.get('perinterval')

            # plot number of operations vs time if per_interval is not empty
            if per_interval:
                # define time samples
                max_time = min(int(duration), len(per_interval))
                time_line = [i for i in range(timeUnit, max_time+timeUnit)]

                plt.figure()
                plt.plot(time_line, per_interval[:len(time_line)],'bo--', label=self.timeplots_added)
                plt.xlabel('time, '+time_unit_literal[4:])
                plt.ylabel('operations per '+time_unit_literal[4:])

                # define file name of the figure
                file = '{0}/plot_per_interval_{1}.png'.format(self.directory, str(self.timeplots_added))
                
                # save figure to file
                plt.savefig(file)
                plt.close()

                # add the file name to the list of files 
                file_names.append(file)

                # increment timeplot count
                self.timeplots_added+=1
        return file_names


