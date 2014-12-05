# fluent-agent-hydra

A Fluentd log agent.

[![Build status](https://api.travis-ci.org/fujiwara/fluent-agent-hydra.svg?branch=master)](https://travis-ci.org/fujiwara/fluent-agent-hydra)

This agent is inspired by [fluent-agent-lite](https://github.com/tagomoris/fluent-agent-lite).

## Features

- Tailing log files (like in_tail)
  - enable to handle multiple files in a single process.
- Forwarding messages to external fluentd (like out_forward)
  - multiple fluentd server can be used. When primary server is down, messages will sent to secondary server.
- Receiving a fluentd's forward protocol messages via TCP (like in_forward)
  - includes simplified on-memory queue.
- Stats monitor httpd server
  - serve an agent stats by JSON format.

## Installation

[Binary releases](https://github.com/fujiwara/fluent-agent-hydra/releases)

or

```
go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-agent-hydra/
```

## Usage

### Using command line arguments

```
fluent-agent-hydra [options] TAG TARGET_FILE PRIMARY_SERVER SECONDARY_SERVER
```

Options

* -f="message": fieldname of fluentd log message attribute (DEFAULT: message)
* -monitor="127.0.0.1:24223": monitor httpd daemon address (DEFAULT: -)

Usage example

```
fluent-agent-hydra -f msg tagname /path/to/foo.log fluentd.example.com:24224 127.0.0.1:24224
```

* Field name: "msg"
* Filename: "/path/to/foo.log"
* Primary server: "fluentd.example.com:24224"
* Secondary server: "127.0.0.1:24224"

### Using configuration file

```
fluent-agent-hydra -c /path/to/config.toml
```

A example of config.toml

```toml
# global settings
TagPrefix = "nginx"       # "nginx.access", "nginx.error"
FieldName = "message"     # default "message"
ReadBufferSize = 1048576  # default 64KB.

# tailing log file (in_tail)
[[Logs]]
File = "/var/log/nginx/access.log"
Tag = "access"

[[Logs]]
File = "/var/log/nginx/error.log"
Tag = "error"

# forwarding fluentd server (out_forward)
[[Servers]]
Host = "fluentd.example.com"
Port = 24224

[[Servers]]
Host = "fluentd-backup.example.com"
Port = 24224

# receive fluentd forward protocol daemon (in_forward)
[Receiver]
Port = 24224

# stats monitor http daemon
[Monitor]
Host = "localhost"
Port = 24223
```

## Stats monitor

### Hydra application stats

`curl -s [Monitor.Host]:[Monitor.Port]/ | jq .`

An example response.

```json
{
  "receiver": {
    "buffered": 0,
    "disposed": 0,
    "messages": 123,
    "max_buffer_messages": 1048576,
    "current_connections": 1,
    "total_connections": 10,
    "address": "[::]:24224"
  },
  "servers": [
    {
      "error": "",
      "alive": true,
      "address": "fluentd.example.com:24224"
    },
    {
      "error": "[2014-08-18 18:25:28.965066394 +0900 JST] dial tcp 192.168.1.11:24224: connection refused",
      "alive": false,
      "address": "fluentd-backup.example.com:24224"
    }
  ],
  "files": {
    "/var/log/nginx/error.log": {
      "error": "",
      "position": 95039,
      "tag": "nginx.error"
    },
    "/var/log/nginx/access.log": {
      "error": "",
      "position": 112093,
      "tag": "nginx.access"
    }
  },
  "sent": {
    "nginx.error": {
      "bytes": 2578,
      "messages": 8
    },
    "nginx.access": {
      "bytes": 44996,
      "messages": 109
    }
  }
}
```

### system stats

`curl -s [Monitor.Host]:[Monitor.Port]/stats | jq .`

An example response.

```json
{
  "time": 1417748153556699400,
  "go_version": "go1.3",
  "go_os": "darwin",
  "go_arch": "amd64",
  "cpu_num": 4,
  "goroutine_num": 17,
  "gomaxprocs": 1,
  "cgo_call_num": 48,
  "memory_alloc": 551840,
  "memory_total_alloc": 17886960,
  "memory_sys": 5310712,
  "memory_lookups": 321,
  "memory_mallocs": 4645,
  "memory_frees": 3622,
  "memory_stack": 131072,
  "heap_alloc": 551840,
  "heap_sys": 2097152,
  "heap_idle": 1253376,
  "heap_inuse": 843776,
  "heap_released": 0,
  "heap_objects": 1023,
  "gc_next": 1083088,
  "gc_last": 1417748153454501600,
  "gc_num": 34,
  "gc_per_second": 0.966939666110773,
  "gc_pause_per_second": 0.641048,
  "gc_pause": [
    0.2991,
    0.341948
  ]
}
```

## Benchmark

* AWS EC2 c3.2xlarge
  * Amazon Linux AMI release 2014.03
  * Linux 3.10.48-55.140.amzn1.x86\_64 #1 SMP Wed Jul 9 23:32:19 UTC 2014 x86\_64 x86\_64 x86\_64 GNU/Linux
* fluentd 0.10.52
  * ruby 2.1.2p95 (2014-05-08 revision 45877) [x86_64-linux]
* fluent-agent-lite 1.0
  * perl This is perl 5, version 16, subversion 3 (v5.16.3) built for x86_64-linux-thread-multi
* fluent-agent-hydra v0.0.5
  * go version go1.3.1 linux/amd64

Benchmark set [fluentd-benchmark/one_forward](https://github.com/fluent/fluentd-benchmark/tree/master/one_forward)

| lines/sec  | fluentd CPU |         RSS | lite CPU |      RSS | hydra(1) CPU |    RSS | hydra(2) CPU | RSS  |
|-----------:|------------:|------------:|---------:|---------:|----------:|----------:|-------------:|-----:|
|        100 |        0.65 |       40152 |     0.18 |     8424 |      0.30 |      5400 |         0.78 |  5244 |
|       1000 |        1.7  |       51836 |     0.34 |     8420 |      0.68 |      5872 |          1.2 |  6212 |
|       5000 |        5.5  |       87316 |      1.0 |     9280 |       1.8 |      7728 |          2.8 |  9804 |
|      10000 |         11  |       85468 |      2.0 |     9812 |       3.4 |      9796 |          5.1 | 10560 |
|      50000 |         48  |      132496 |      8.9 |    15216 |        15 |     10840 |           24 | 11728 |
|     100000 |        107  |      636892 |       18 |    20948 |        31 |     11092 |           48 | 13284 |
|     250000 |          -  |           - |       45 |    21048 |        71 |     15748 |          115 | 20076 |
|     500000 |          -  |           - |       87 |    21092 |     87(a) |  65788(a) |          173 | 30344 |
|     700000 |          -  |           - |       -  |       -  |         - |         - |       165(b) | 81756(b) |

* fluent-agent-lite max:  580,000/sec
* fluent-agent-hydra(GOMAXPROCS=1) max: 460,000/sec
* fluent-agent-hydra(GOMAXPROCS=1, ReadBufferSize=1,000,000) max: 550,000/sec (a)
* fluent-agetn-hydra(GOMAXPROCS=2, ReadBufferSize=1,000,000) max: 700,000/sec (b)

## Thanks to

* `fluent/fluent.go, utils.go` imported and modified from [github.com/t-k/fluent-logger-golang](https://github.com/t-k/fluent-logger-golang).
* `fluent/server.go` imported and modified from [github.com/moriyoshi/ik](https://github.com/moriyoshi/ik/).

## Author

Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

## Licence

Copyright 2014 Fujiwara Shunichiro.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
