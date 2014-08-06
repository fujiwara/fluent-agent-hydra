package hydra

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
)

type Stats struct {
	Sent   map[string]*Stat `json:"sent"`
	mu     sync.Mutex
}

type Stat struct {
	Tag      string `json:"-"`
	Messages int64  `json:"messages"`
	Bytes    int64  `json:"bytes"`
}

func (ss *Stats) Add(s *Stat) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if _s, ok := ss.Sent[s.Tag]; ok {
		_s.Messages += s.Messages
		_s.Bytes += s.Bytes
	} else {
		ss.Sent[s.Tag] = &Stat{
			Tag:      s.Tag,
			Messages: s.Messages,
			Bytes:    s.Bytes,
		}
	}
}

func (ss *Stats) WriteJSON(w http.ResponseWriter) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	encoder := json.NewEncoder(w)
	encoder.Encode(ss)
}

func (ss *Stats) RecieveStat(ch chan *Stat) {
	for {
		s := <-ch
		ss.Add(s)
	}
}

func NewMonitorServer(ch chan *Stat, addr string) (net.Addr, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	ss := &Stats{
		Sent: make(map[string]*Stat),
	}
	go ss.RecieveStat(ch)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ss.WriteJSON(w)
	})
	go http.Serve(listener, nil)
	log.Printf("[info] Monitor server listening http://%s/\n", listener.Addr())
	return listener.Addr(), err
}
