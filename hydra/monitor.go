package hydra

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
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
	if _f, ok := ss.Files[s.File]; ok {
		_f.Position = s.Position
		_f.Error = s.Error
	} else {
		ss.Files[s.File] = s
	}
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

func (ss *Stats) RecieveStat(ch chan Stat) {
	for {
		s := <-ch
		s.ApplyTo(ss)
	}
}

func MonitorServer(config *Config, monitorCh chan Stat) (net.Addr, error) {
	ss := &Stats{
		Sent:    make(map[string]*SentStat),
		Files:   make(map[string]*FileStat),
		Servers: make([]*ServerStat, len(config.Servers)),
	}
	go ss.RecieveStat(monitorCh)

	if config.MonitorAddress == "" {
		return nil, nil
	}
	listener, err := net.Listen("tcp", config.MonitorAddress)
	if err != nil {
		return nil, err
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ss.WriteJSON(w)
	})
	go http.Serve(listener, nil)
	log.Printf("[info] Monitor server listening http://%s/\n", listener.Addr())
	return listener.Addr(), err
}
