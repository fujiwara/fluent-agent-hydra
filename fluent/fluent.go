/*
This package was imported from https://github.com/t-k/fluent-logger-golang and modified.

Original License:

Copyright (c) 2013 Tatsuo Kaniwa

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fluent

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

const (
	defaultHost                   = "127.0.0.1"
	defaultPort                   = 24224
	defaultTimeout                = 3 * time.Second
	defaultRetryWait              = 500
	defaultMaxRetry               = 12
	defaultReconnectWaitIncreRate = 1.5
)

type Config struct {
	FluentPort int
	FluentHost string
	Timeout    time.Duration
	RetryWait  int
	MaxRetry   int
}

type Fluent struct {
	Config
	conn         net.Conn
	pending      []byte
	reconnecting bool
	mu           sync.Mutex
}

// New creates a new Logger.
func New(config Config) (f *Fluent, err error) {
	if config.FluentHost == "" {
		config.FluentHost = defaultHost
	}
	if config.FluentPort == 0 {
		config.FluentPort = defaultPort
	}
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}
	if config.RetryWait == 0 {
		config.RetryWait = defaultRetryWait
	}
	if config.MaxRetry == 0 {
		config.MaxRetry = defaultMaxRetry
	}
	f = &Fluent{Config: config, reconnecting: false}
	err = f.connect()
	return
}

// PostBulkMessages post multi messages by packed one MessagePack object.
func (f *Fluent) PostBulkMessages(tag string, key string, messages [][]byte) error {
	timeUnix := time.Now().Unix()
	buffer := make([]byte, 0, len(messages)*1024)
	for _, message := range messages {
		msg := []interface{}{tag, timeUnix, map[string][]byte{key: message}}
		if data, dumperr := toMsgpack(msg); dumperr != nil {
			fmt.Println("fluent#Post: Can't convert to msgpack:", msg, dumperr)
			return dumperr
		} else {
			buffer = append(buffer, data...)
		}
	}
	if err := f.send(buffer); err != nil {
		f.Close()
		return err
	}
	return nil
}

// Close closes the connection.
func (f *Fluent) Close() (err error) {
	if f.conn != nil {
		f.mu.Lock()
		defer f.mu.Unlock()
	} else {
		return
	}
	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}
	return
}

// FluentAddr return fluentd address string e.g. "127.0.0.1:24224"
func (f *Fluent) FluentdAddr() string {
	return fmt.Sprintf("%s:%d", f.Config.FluentHost, f.Config.FluentPort)
}

// IsReconnecting return true if a reconnecting process in progress.
func (f *Fluent) IsReconnecting() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reconnecting
}

// connect establishes a new connection using the specified transport.
func (f *Fluent) connect() (err error) {
	f.conn, err = net.DialTimeout("tcp", f.FluentdAddr(), f.Config.Timeout)
	return
}

func (f *Fluent) reconnect() {
	for i := 0; ; i++ {
		err := f.connect()
		if err == nil {
			f.mu.Lock()
			f.reconnecting = false
			f.mu.Unlock()
			log.Println("Successfully reconnected to", f.FluentdAddr())
			break
		} else {
			waitN := math.Min(float64(i), float64(f.Config.MaxRetry))
			waitTime := f.Config.RetryWait * e(defaultReconnectWaitIncreRate, waitN)
			log.Printf("Waiting %.1f sec to reconnect %s", float64(waitTime)/float64(1000), f.FluentdAddr())
			time.Sleep(time.Duration(waitTime) * time.Millisecond)

		}
	}
}

func (f *Fluent) send(buffer []byte) (err error) {
	if f.conn == nil {
		log.Println("send() but conn is nil")
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.reconnecting == false {
			log.Println("going to reconnect")
			f.reconnecting = true
			go f.reconnect()
		}
		err = errors.New("Can't send logs, client is reconnecting")
	} else {
		_, err = f.conn.Write(buffer)
	}
	return
}

