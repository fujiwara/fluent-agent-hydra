# fluent-agent-hydra

A Fluentd log agent.

[![](https://github.com/fujiwara/fluent-agent-hydra/workflows/Go/badge.svg)](https://github.com/fujiwara/fluent-agent-hydra/actions?query=workflow%3AGo+branch%3Amaster)
[![](https://github.com/fujiwara/fluent-agent-hydra/workflows/Release/badge.svg)](https://github.com/fujiwara/fluent-agent-hydra/actions?query=workflow%3ARelease+branch%3Amaster)

This agent is inspired by [fluent-agent-lite](https://github.com/tagomoris/fluent-agent-lite).

## Features

- Tailing log files (like in_tail)
  - enable to handle multiple files in a single process.
  - parse JSON or LTSV format.
- Forwarding messages to external fluentd (like out_forward)
  - multiple fluentd server can be used. When primary server is down, messages will sent to secondary server.
  - if config.ServerRoundRobin = true, select one server from all servers by round robin.
- Receiving a fluentd's forward protocol messages via TCP (like in_forward)
  - includes simplified on-memory queue.
- Stats monitor httpd server
  - serve an agent stats by JSON format.
- Supports sub-second time
  - Supported by Fluentd 0.14 or later. When you use Fluentd <= 0.12 as forwarded servers, fluentd will not accept records including sub-second time.

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
ServerRoundRobin = true   # default false
SubSecondTime = true      # default false. for Fluentd 0.14 or later only

# tailing log file (in_tail)
[[Logs]]
File = "/var/log/nginx/access.log"
Tag = "access"
# parse as ltsv format. (see http://ltsv.org/)
# Format = "None"(default) | "LTSV" | "JSON" | "Regexp"
Format = "LTSV"

# If Format is "Regexp", Regexp directive is required.
# Regexp = "(your regexp string)" | "apache" | "nginx" | "syslog"

# convert column data type
# 'column1_name:type,column2_name:type'
# type = "interger" | "float" | "bool" | otherwise as string
Types = "reqtime:float,size:integer,apptime:float,status:integer"

# parse a time string in log lines, and set it as record's timestamp
TimeParse = true      # default false
TimeKey = "timestamp" # default "time"

# TimeFormat is passed to Golang's time.Parse().
# http://golang.org/pkg/time/#Parse
# default time.RFC3339 == "2006-01-02T15:04:05Z07:00"
# "apache" | "nginx" | "syslog" | "unix" is also available
TimeFormat = "02/Jan/2006:15:04:05 Z0700"

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

### About special conversion behavior for numerical value

When the `Format` is JSON, fluent-agent-hydra treats a numerical value as float64 even if its type is integer.
For treating a numerical value as integer, set a column data type integer with the `Types`.

```toml
Types = "column_name:integer"
```

Its type is converted to int64.

## Stats monitor

For enabling stats monitor, specify command line option `-m host:port` or `[Monitor]` section in config file.

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

`curl -s [Monitor.Host]:[Monitor.Port]/system | jq .`

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

See [benchmark/README](benchmark/README.md) .

## Thanks to

* `fluent/fluent.go, utils.go` imported and modified from [github.com/t-k/fluent-logger-golang](https://github.com/t-k/fluent-logger-golang).
* `fluent/server.go` imported and modified from [github.com/moriyoshi/ik](https://github.com/moriyoshi/ik/).

## Author

Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

## Licence

Copyright 2014 Fujiwara Shunichiro. / KAYAC Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
