package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/ltsv"
)

var (
	MessageCh        = make(map[string]map[int64]chan fluent.FluentRecordSet)
	MessageBufferLen = 100
	MessageChMutex   sync.Mutex
	ConnectionId     int64
	Counter          int64
	DumpCh           chan fluent.FluentRecordSet
)

type OutputOption struct {
	IncludeTime bool
	IncludeTag  bool
}

type Encoder interface {
	Encode(interface{}) error
}

func main() {
	var (
		forwardPort     int
		httpPort        int
		countInterval   int
		dir             string
		dumpIncludeTag  bool
		dumpIncludeTime bool
		dumpFileFormat  string
	)
	flag.IntVar(&forwardPort, "forward-port", 24224, "fluentd forward listen port")
	flag.IntVar(&httpPort, "http-port", 24225, "http listen port")
	flag.IntVar(&countInterval, "count-interval", 60, "log counter output interval(sec)")
	flag.StringVar(&dir, "dump-file-dir", "", "dump file directory")
	flag.BoolVar(&dumpIncludeTag, "dump-include-tag", false, "dump file include tag")
	flag.BoolVar(&dumpIncludeTime, "dump-include-time", false, "dump file include time")
	flag.StringVar(&dumpFileFormat, "dump-file-format", "json", "dump file format")
	flag.Parse()

	go runReporter(time.Duration(countInterval) * time.Second)
	if dir != "" {
		DumpCh = make(chan fluent.FluentRecordSet, MessageBufferLen)
		option := OutputOption{dumpIncludeTime, dumpIncludeTag}
		go runDumper(dir, dumpFileFormat, option)
	}
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

func runDumper(dir string, format string, option OutputOption) {
	var filename string
	var fh io.WriteCloser
	var err error
	for {
		rs := <-DumpCh
		now := time.Now().Format("2006-01-02-15")
		_filename := filepath.Join(dir, rs.Tag+"."+now)
		if filename != _filename {
			if fh != nil {
				fh.Close()
			}
			filename = _filename
			fh, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
			defer fh.Close()
			if err != nil {
				log.Println("[error]", err)
				continue
			}
		}
		if fh == nil {
			continue
		}
		var encoder Encoder
		switch format {
		case "ltsv":
			encoder = ltsv.NewEncoder(fh)
		default:
			encoder = json.NewEncoder(fh)
		}
		for _, record := range rs.Records {
			record, _ := record.(*fluent.TinyFluentRecord)
			if option.IncludeTime {
				fmt.Fprint(fh, record.Timestamp.Format(time.RFC3339), "\t")
			}
			if option.IncludeTag {
				fmt.Fprint(fh, rs.Tag, "\t")
			}
			err = encoder.Encode(record.GetAllData())
			if err != nil {
				continue
			}
		}
	}
}

func runReporter(t time.Duration) {
	ticker := time.Tick(t)
	for _ = range ticker {
		c := atomic.SwapInt64(&Counter, 0)
		if c > 0 {
			log.Println("count:", c, "cps:", float64(c)/float64(t/time.Second))
		}
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
			atomic.AddInt64(&Counter, int64(len(recordSet.Records)))
			if DumpCh != nil {
				select {
				case DumpCh <- recordSet:
				default:
					log.Printf("[warn] %d records dropped for dump file. tag: %s", len(recordSet.Records), recordSet.Tag)
				}
			}
			for tag, channels := range MessageCh {
				if !matchTag(tag, recordSet.Tag) {
					continue
				}
				for _, ch := range channels {
					select {
					case ch <- recordSet:
					default:
						log.Printf("[warn] %d records dropped for http client.", len(recordSet.Records))
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
		encoder = ltsv.NewEncoder(w)
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
			fmt.Fprint(w, record.Timestamp.Format(time.RFC3339), "\t")
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
