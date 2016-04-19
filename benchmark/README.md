# Benchmark

* AWS EC2 c4.4xlarge
  * Amazon Linux AMI release 2016.03
  * Linux 4.1.10-17.31.amzn1.x86_64
* fluentd 0.12.22
  * ruby 2.2.4p230 (2015-12-16 revision 53155) [x86_64-linux-musl]
* fluent-agent-hydra v0.2.0
  * go version go1.6 linux/amd64

## Benchmark set

format: None, Regexp(apache)
```
192.168.0.1 - - [28/Feb/2013:12:00:00 +0900] "GET / HTTP/1.1" 200 777 "-" "Opera/12.0"
```

format: LTSV
```
time:2013-02-28T12:00:00Z+09:00	host:192.168.0.1	user:-	method:GET	path:/	code:200	size:777	referer:-	agent:Opera/12.0
```

format: JSON
```
{"time":"2013-02-28T12:00:00Z+09:00","host":"192.168.0.1","user":"-","method":"GET","path":"/","code":200,"size":777,"referer":"-","agent":"Opera/12.0"}
```

- [hydra.conf](hydra/hydra.conf)
- [fluentd.conf](fluentd/fluentd.conf)

## Result

### format None

| lines/sec  | hydra CPU  | hydra RSS   | fluentd CPU | fluentd RSS |
|-----------:|-----------:|------------:|------------:|------------:|
|       1000 |       1.42 |       11436 |        0.84 |       38652 |
|       5000 |       2.12 |       13228 |        3.56 |       49072 |
|      10000 |       3.47 |       15872 |        6.64 |       53404 |
|      50000 |       13.2 |       15872 |        34.6 |       57368 |
|     100000 |       25.2 |       15872 |        69.3 |       61688 |

### format LTSV

| lines/sec  | hydra CPU  | hydra RSS   | fluentd CPU | fluentd RSS |
|-----------:|-----------:|------------:|------------:|------------:|
|       1000 |       2.36 |       12216 |        3.03 |       43936 |
|       5000 |        7.9 |       12772 |        14.3 |       45388 |
|      10000 |       16.0 |       13868 |        28.7 |       48156 |
|      50000 |       80.3 |       14368 |         100 |       49732 |
|     100000 |        158 |       14888 |         N/A |         N/A |

* fluentd sent max 35,103/sec
* hydra sent max 104,795/sec

### format JSON

| lines/sec  | hydra CPU  | hydra RSS   | fluentd CPU | fluentd RSS |
|-----------:|-----------:|------------:|------------:|------------:|
|       1000 |       2.99 |       12728 |        2.76 |       49948 |
|       5000 |       11.2 |       12728 |        11.8 |       53500 |
|      10000 |       21.8 |       13180 |        24.0 |       52500 |
|      50000 |        102 |       13704 |         100 |       55756 |
|     100000 |        141 |       13964 |         N/A |         N/A |

* fluentd sent max 43,235/sec
* hydra sent max  70,222/sec

### format Regexp(apache)

| lines/sec  | hydra CPU  | hydra RSS   | fluentd CPU | fluentd RSS |
|-----------:|-----------:|------------:|------------:|------------:|
|       1000 |       2.56 |       12676 |        1.79 |       47784 |
|       5000 |       8.93 |       12676 |        8.61 |       51996 |
|      10000 |       17.0 |       13756 |        17.2 |       53432 |
|      50000 |       90.0 |       13756 |        87.2 |       55948 |
|     100000 |        157 |       13984 |        N/A  |         N/A |

* fluentd sent max 52,905/sec
* hydra sent max 88,887/sec

## To get more performance

### Tune `GOGC`

Set a large number to `GOGC` environment variable. (e.g. `GOGC=1000`)

hydra will eat more memories, but you will get more peak througput.

format: Regexp(apache)

| GOGC          | peak lines/sec  |  RSS    |
|---------------|----------------:|--------:|
| 100 (default) |          88,887 |  13984  |
| 1000          |         11,0841 |  51868  |
