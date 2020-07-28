# LogBench Simple Log Agent Benchmarker

## How to build
```
go build github.com/awslabs/amazon-log-agent-benchmark-tool/cmd/logbench/
```

## How to use
logbench can be used with the following command

```
./logbench -log LOGFILE1,LOGFILE2 -log LOGFILE3 COMMAND PARAM1 PARAM2

  -f duration
        Frequency to collect metrics represented in time duration, default 1s (default 1s)
  -line string
        Content of the log line to be used (default "INFO CloudWatchOutput      Amazon::Monitoring::CloudWatchOutput::new - CloudWatchOutput sender=data/cloudwatch/current endpoint=https://monitoring.us-east-1.amazonaws.com maxBytes=76800")
  -log value
        Path of the log files being generated and writes logs to, you can specify multiple values by using the parameter multiple times or use comma seperated list
  -multilinestart string
        Regular expression of a start of a multiline log event
  -o    Pipe agent output to stdout and stderr
  -p int
        Pid of the agent to check resource usage (default -1)
  -r duration
        Ramp up duration, time for agent to stablize, stats will not be collected during the ramp up, default 1s (default 1s)
  -rate value
        Log generation rate to be tested, e.g. -log 1,100,1k,10k,100k, default 100
  -replay string
        Path to a file for log replay
  -replaytimelayout string
        Format to parse and replace the timestamp for replaying log file, e.g. -replaytimelayout='Mon, 02 Jan 2006 15:04:05 MST', or use a capture group like -replaytimelayout='"timestamp":"(2006-01-02T15:04:05-0700)"' following Go time layout, see: https://golang.org/pkg/time/#pkg-constants
  -rotatekeep int
        Number of rotation files to keep, 0 to disable rotation
  -rotatesize string
        Size of the logfile before rotation
  -rotatetime duration
        How much time the logfile should be rotated
  -t duration
        Test duration, in format supported by time.ParseDuration, default 10s (default 10s)
  -timelayout string
        Format to print the timestamp for the log lines, following Go time layout, see: https://golang.org/pkg/time/#pkg-constants (default "Jan _2 15:04:05.000000000")
```

Example usage:
```
logbench -log stream1.log,stream2.log -rate 100,1000,10000 -t 50s -r 5s -f 2s ./amazon-cloudwatch-agent -config test.conf
```

This would generate 2 log files stream1.log and stream2.log, start the agent with the command `./amazon-cloudwatch-agent -config test.conf` at a rate of 100 lines/s, 1000 lines/s, and 10000 lines/s each for 5s first, then collect 50s of cpu and memory usage data every 2s.
At the end of the benchmark, the agent will be sent with SIGINT first, if it does not exit in 5 seconds, the agent process would be killed.

Replay log file:
```
logbench -log test.log -replay=original.log -replaytimelayout='Mon, 02 Jan 2006 15:04:05 MST'
```
This would replay the log file, using the replaytimelayout time layout to match timestamp from the log and replace the timestamp with current time. Lines are output based on the original delay between log lines from the source log file.
