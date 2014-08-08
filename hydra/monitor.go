package hydra

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Stats struct {
	Sent    map[string]*SentStat `json:"sent"`
	Files   map[string]*FileStat `json:"files"`
	Servers []*ServerStat        `json:"servers"`
	mu      sync.Mutex
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
}

type FileStat struct {
	Tag      string `json:"tag"`
	File     string `json:"-"`
	Position int64  `json:"position"`
	Error    string `json:"error"`
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
	} else {
		ss.Sent[s.Tag] = s
	}
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

func NewMonitor(config *Config, monitorCh chan Stat) (*Monitor, error) {
	stats := &Stats{
		Sent:    make(map[string]*SentStat),
		Files:   make(map[string]*FileStat),
		Servers: make([]*ServerStat, len(config.Servers)),
	}
	go stats.Run(monitorCh)
	monitor := &Monitor{
		stats:     stats,
		address:   config.MonitorAddress,
		monitorCh: monitorCh,
	}
	if monitor.address == "" {
		return monitor, nil
	}
	listener, err := net.Listen("tcp", monitor.address)
	if err != nil {
		log.Println("[error]", err)
		return nil, err
	}
	monitor.listener = listener
	monitor.Addr = listener.Addr()
	return monitor, nil
}

func (m *Monitor) Run() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m.stats.WriteJSON(w)
	})
	go http.Serve(m.listener, nil)
	log.Printf("[info] Monitor server listening http://%s/\n", m.listener.Addr())
}

func monitorError(err error) string {
	return fmt.Sprintf("[%s] %s", time.Now(), err)
}
