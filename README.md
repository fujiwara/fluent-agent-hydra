# fluent-agent-hydra

A Fluentd log agent.

[![Build status](https://api.travis-ci.org/fujiwara/fluent-agent-hydra.svg?branch=master)](https://travis-ci.org/fujiwara/fluent-agent-hydra)

## Installation

```
go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-agent-hydra/
```

## Usage

Usage of fluent-agent-hydra

### Using configuration file

```
fluent-agent-hydra -c /path/to/config.toml
```

A example of config.toml

```toml
Servers = [ "127.0.0.1:24224", "127.0.0.1:24225" ]
TagPrefix = "foo"
FieldName = "msg"

[[Logs]]
Tag  = "tag1"
File = "/path/to/foo.log"

[[Logs]]
Tag  = "tag2"
File = "/path/to/bar.log"
```

### Using command line arguments

```
fluent-agent-hydra [options] TAG TARGET_FILE PRIMARY_SERVER SECONDARY_SERVER
```

Options

* -f="message": fieldname of fluentd log message attribute (DEFAULT: message)

Usage example

```
fluent-agent-hydra -f msg tagname /path/to/foo.log fluentd.example.com:24224 127.0.0.1:24224
```

* Field name: "msg"
* Filename: "/path/to/foo.log"
* Primary server: "fluentd.example.com:24224"
* Secondary server: "127.0.0.1:24224"

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
