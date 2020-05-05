# LogBench Simple Log Agent Benchmarker

## How to build
```
go build github.com/awslabs/amazon-log-agent-benchmark-tool/cmd/logbench/
```

## How to use
LogBench acceptes the follow parameters:

```
  -f duration
        Frequency to collect metrics represented in time duration, default 1s (default 1s)
  -log value
        Path of the log files being generated and writes logs to, you can specify multiple values by using the parameter multiple times or use comma seperated list.
  -r duration
        Ramp up duration, time for agent to stablize, stats will not be collected during the ramp up, default 1s (default 1s)
  -rate value
        Log generation rate to be tested, e.g. -log 1,100,1k,10k,100k, default 100
  -t duration
        Test duration, in format supported by time.ParseDuration, default 10s (default 10s)
```

Example usage:
```
LogBench -log stream1.log,stream2.log -rate 100,1000,10000 -t 50s -r 5s -f 2s ./amazon-cloudwatch-agent -config test.conf
```

This would generate 2 log files stream1.log and stream2.log, start the agent with the command `./amazon-cloudwatch-agent -config test.conf` at a rate of 100 lines/s, 1000 lines/s, and 10000 lines/s each for 5s first, then collect 50s of cpu and memory usage data every 2s.
At the end of the benchmark, the agent will be sent with SIGINT first, if it does not exit in 5 seconds, the agent process would be killed.

