package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

var (
	MessageCh        = make(map[string]map[int64]chan fluent.FluentRecordSet)
	MessageBufferLen = 100
	MessageChMutex   sync.Mutex
	ConnectionId     int64
)

type OutputOption struct {
	IncludeTime bool
	IncludeTag  bool
}

type Encoder interface {
	Encode(interface{}) error
}

type LTSVEncoder struct {
	writer io.Writer
}

func NewLTSVEncoder(w io.Writer) *LTSVEncoder {
	return &LTSVEncoder{writer: w}
}

func (w *LTSVEncoder) Encode(data interface{}) error {
	first := true
	switch record := data.(type) {
	default:
		return fmt.Errorf("unsupported type")
	case map[string]interface{}:
		for key, value := range record {
			if !first {
				_, err := fmt.Fprint(w.writer, "\t")
				if err != nil {
					return err
				}
			} else {
				first = false
			}
			_, err := fmt.Fprint(w.writer, key, ":", value)
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(w.writer, "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var (
		forwardPort int
		httpPort    int
	)
	flag.IntVar(&forwardPort, "forward-port", 24224, "fluentd forward listen port")
	flag.IntVar(&httpPort, "http-port", 24225, "http listen port")
	flag.Parse()

	var err error
	err = runForwardServer(forwardPort)
	if err != nil {
		log.Fatal(err)
	}
	err = runHTTPServer(httpPort)
	if err != nil {
		log.Fatal(err)
	}
}

func runForwardServer(port int) error {
	addr := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Println("[info] forward server listing", l.Addr())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("[error] accept error", err)
			}
			go handleForwardConn(conn)
		}
	}()
	return nil
}

func handleForwardConn(conn net.Conn) {
	defer conn.Close()
	for {
		recordSets, err := fluent.DecodeEntries(conn)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Println("decode entries failed", err, conn.LocalAddr())
			return
		}
		for _, recordSet := range recordSets {
			for tag, channels := range MessageCh {
				if !matchTag(tag, recordSet.Tag) {
					continue
				}
				for _, ch := range channels {
					select {
					case ch <- recordSet:
					}
				}
			}
		}
	}
}

func runHTTPServer(port int) error {
	http.HandleFunc("/", httpHandler)
	addr := fmt.Sprintf(":%d", port)
	log.Println("[info] http server listing", addr)
	return http.ListenAndServe(addr, nil)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	tag := strings.Trim(r.URL.Path, "/")
	if tag == "" {
		http.NotFound(w, r)
		return
	}

	option := OutputOption{false, false}
	option.IncludeTime, _ = strconv.ParseBool(r.FormValue("time"))
	option.IncludeTag, _ = strconv.ParseBool(r.FormValue("tag"))
	var encoder Encoder
	switch t, _ := strconv.ParseBool(r.FormValue("ltsv")); t {
	case true:
		encoder = NewLTSVEncoder(w)
	default:
		encoder = json.NewEncoder(w)
	}

	id := atomic.AddInt64(&ConnectionId, 1)
	ch := subscribe(tag, id)
	defer unsubscribe(tag, id)

	log.Printf(
		"[info] client %s tag:%s",
		r.RemoteAddr,
		tag,
	)

	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	for {
		recordSet := <-ch
		err := writeResponse(encoder, w, recordSet, option)
		if err != nil {
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	return
}

func matchTag(matcher, tag string) bool {
	matcher = strings.TrimRight(matcher, ".*")
	return strings.HasPrefix(tag, matcher)
}

func subscribe(tag string, id int64) chan fluent.FluentRecordSet {
	MessageChMutex.Lock()
	defer MessageChMutex.Unlock()

	if m := MessageCh[tag]; m == nil {
		MessageCh[tag] = make(map[int64]chan fluent.FluentRecordSet)
	}
	ch := make(chan fluent.FluentRecordSet, MessageBufferLen)
	MessageCh[tag][id] = ch
	return ch
}

func unsubscribe(tag string, id int64) {
	MessageChMutex.Lock()
	defer MessageChMutex.Unlock()

	delete(MessageCh[tag], id)
}

func writeResponse(encoder Encoder, w http.ResponseWriter, rs fluent.FluentRecordSet, option OutputOption) error {
	for _, record := range rs.Records {
		record, _ := record.(*fluent.TinyFluentRecord)
		data := record.GetAllData()
		if option.IncludeTime {
			fmt.Fprint(w, time.Unix(record.Timestamp, 0).Format(time.RFC3339), "\t")
		}
		if option.IncludeTag {
			fmt.Fprint(w, rs.Tag, "\t")
		}
		err := encoder.Encode(data)
		if err != nil {
			return err
		}
	}
	return nil
}
