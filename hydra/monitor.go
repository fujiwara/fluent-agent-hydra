package hydra

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	stats_api "github.com/fukata/golang-stats-api-handler"
)

const (
	DefaultMonitorPort = 24223
	DefaultMonitorHost = "localhost"
)

type Stats struct {
	Sent     map[string]*SentStat `json:"sent"`
	Files    map[string]*FileStat `json:"files"`
	Servers  []*ServerStat        `json:"servers"`
	Receiver *ReceiverStat        `json:"receiver"`
	mu       sync.Mutex
}

type Stat interface {
	ApplyTo(*Stats)
}

type ServerStat struct {
	Index   int    `json:"-"`
	Address string `json:"address"`
	Alive   bool   `json:"alive"`
	Error   string `json:"error"`
}

type SentStat struct {
	Tag      string `json:"-"`
	Messages int64  `json:"messages"`
	Bytes    int64  `json:"bytes"`
	Sents    int64  `json:"sents"`
}

type FileStat struct {
	Tag      string `json:"tag"`
	File     string `json:"-"`
	Position int64  `json:"position"`
	Error    string `json:"error"`
}

type ReceiverStat struct {
	Address            string `json:"address"`
	Connections        int    `json:"-"`
	TotalConnections   int    `json:"total_connections"`
	CurrentConnections int    `json:"current_connections"`
	Messages           int64  `json:"messages"`
	Disposed           int64  `json:"disposed"`
	Buffered           int64  `json:"buffered"`
	MaxBufferMessages  int64  `json:"max_buffer_messages"`
}

func (s *FileStat) ApplyTo(ss *Stats) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.Files[s.File] = s
}

func (s *ServerStat) ApplyTo(ss *Stats) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.Servers[s.Index] = s
}

func (s *SentStat) ApplyTo(ss *Stats) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if _s, ok := ss.Sent[s.Tag]; ok {
		_s.Messages += s.Messages
		_s.Bytes += s.Bytes
		_s.Sents += s.Sents
	} else {
		ss.Sent[s.Tag] = s
	}
}

func (s *ReceiverStat) ApplyTo(ss *Stats) {
	if ss.Receiver == nil {
		ss.Receiver = s
		return
	}
	rs := ss.Receiver
	if s.Address != "" {
		rs.Address = s.Address
	}
	if s.Connections > 0 {
		rs.TotalConnections += s.Connections
	}
	rs.CurrentConnections += s.Connections
	rs.Messages += s.Messages
	rs.Disposed += s.Disposed
	rs.Buffered = s.Buffered
}

func (ss *Stats) WriteJSON(w http.ResponseWriter) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	encoder := json.NewEncoder(w)
	encoder.Encode(ss)
}

func (ss *Stats) Run(ch chan Stat) {
	for {
		s := <-ch
		s.ApplyTo(ss)
	}
}

type Monitor struct {
	stats     *Stats
	address   string
	Addr      net.Addr
	listener  net.Listener
	monitorCh chan Stat
}

func NewMonitor(config *Config) (*Monitor, error) {
	stats := &Stats{
		Sent:    make(map[string]*SentStat),
		Files:   make(map[string]*FileStat),
		Servers: make([]*ServerStat, len(config.Servers)),
	}
	monitor := &Monitor{
		stats: stats,
	}
	if config.Monitor == nil {
		return monitor, nil
	}
	monitorAddress := fmt.Sprintf("%s:%d", config.Monitor.Host, config.Monitor.Port)
	listener, err := net.Listen("tcp", monitorAddress)
	if err != nil {
		log.Println("[error]", err)
		return nil, err
	}
	monitor.listener = listener
	monitor.Addr = listener.Addr()
	return monitor, nil
}

func (m *Monitor) Run(c *Context) {
	c.OutputProcess.Add(1)
	defer c.OutputProcess.Done()
	go m.stats.Run(c.MonitorCh)

	c.StartProcess.Done()

	if m.listener == nil {
		return
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m.stats.WriteJSON(w)
	})
	http.HandleFunc("/system", stats_api.Handler)

	go http.Serve(m.listener, nil)
	log.Printf("[info] Monitor server listening http://%s/\n", m.listener.Addr())
}

func monitorError(err error) string {
	return fmt.Sprintf("[%s] %s", time.Now(), err)
}
