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
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	defaultServer                 = "127.0.0.1:24224"
	defaultTimeout                = 3 * time.Second
	defaultRetryWait              = 500
	defaultMaxRetry               = 12
	defaultReconnectWaitIncreRate = 1.5
	debug                         = false
)

type Config struct {
	Server    string
	Timeout   time.Duration
	RetryWait int
	MaxRetry  int
}

type Fluent struct {
	Config
	conn            net.Conn
	pending         []byte
	reconnecting    bool
	cancelReconnect chan bool
	mu              sync.Mutex
	lastError       error
	lastErrorAt     time.Time
	Sent            int64
}

var Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// New creates a new Logger.
func New(config Config) (f *Fluent, err error) {
	if config.Server == "" {
		config.Server = defaultServer
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
	f = &Fluent{
		Config:          config,
		reconnecting:    false,
		cancelReconnect: make(chan bool),
	}
	err = f.connect()
	return
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

func (f *Fluent) Shutdown() {
	if f.IsReconnecting() {
		close(f.cancelReconnect)
	}
	f.Close()
}

func (f *Fluent) String() string {
	var state string
	if f.IsReconnecting() {
		state = "reconnecting"
	} else {
		state = "connected"
	}
	return fmt.Sprintf("*fluent.Fluent{server: '%s', state: '%s'}", f.Server, state)
}

// IsReconnecting return true if a reconnecting process in progress.
func (f *Fluent) IsReconnecting() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reconnecting
}

func (f *Fluent) Alive() bool {
	return f.conn != nil
}

// connect establishes a new connection using the specified transport.
func (f *Fluent) connect() (err error) {
	host, port, err := net.SplitHostPort(f.Server)
	if err != nil {
		return err
	}
	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		return err
	}
	// for DNS round robin
	n := Rand.Intn(len(addrs))
	addr := addrs[n]
	var format string
	if strings.Contains(addr, ":") {
		// v6
		format = "[%s]:%s"
	} else {
		// v4
		format = "%s:%s"
	}
	resolved := fmt.Sprintf(format, addr, port)
	log.Printf("[info] Connect to %s (%s)", f.Server, resolved)
	f.conn, err = net.DialTimeout("tcp", resolved, f.Config.Timeout)
	f.recordError(err)
	return
}

func (f *Fluent) recordError(err error) {
	f.lastErrorAt = time.Now()
	f.lastError = err
}

func (f *Fluent) reconnect() {
	log.Println("[info] Trying reconnect to", f.Server)
	for i := 0; ; i++ {
		err := f.connect()
		if err == nil {
			f.mu.Lock()
			f.reconnecting = false
			f.recordError(err)
			f.mu.Unlock()
			log.Println("[info] Successfully reconnected to", f.Server)
			return
		} else {
			log.Println("[warning]", err)
		}
		waitN := math.Min(float64(i), float64(f.Config.MaxRetry))
		waitTime := f.Config.RetryWait * e(defaultReconnectWaitIncreRate, waitN)
		log.Printf("[info] Waiting %.1f sec to reconnect %s", float64(waitTime)/float64(1000), f.Server)

		select { // wait for timeout or cancel
		case _, ok := <-f.cancelReconnect:
			if !ok {
				f.mu.Lock()
				f.reconnecting = false
				f.mu.Unlock()
				log.Println("[info] Accept cancel reconnect")
				return
			}
		case <-time.After(time.Duration(waitTime) * time.Millisecond):
		}
	}
}

func (f *Fluent) RefreshConnection() error {
	f.Close()
	if err := f.connect(); err != nil {
		go f.reconnect()
		return err
	}
	return nil
}

func (f *Fluent) Send(buffer []byte) (err error) {
	if f.conn == nil {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !f.reconnecting {
			f.reconnecting = true
			go f.reconnect()
		}
		err = errors.New("Can't send messages, client is reconnecting")
		f.recordError(err)
		return
	} else {
		f.conn.SetWriteDeadline(time.Now().Add(f.Config.Timeout))
		_, err = f.conn.Write(buffer)
		if err != nil {
			f.recordError(err)
			f.Close()
		}
		f.Sent++
	}
	return
}

func (f *Fluent) LastErrorString() string {
	if f.lastError != nil {
		return fmt.Sprintf("[%s] %s", f.lastErrorAt, f.lastError)
	} else {
		return ""
	}
}

func e(x, y float64) int {
	return int(math.Pow(x, y))
}
