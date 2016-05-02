package hydra

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"gopkg.in/fsnotify.v1"
)

const (
	TailInterval = 200 * time.Millisecond
)

type InTail struct {
	filename       string
	tag            string
	fieldName      string
	lastReadAt     time.Time
	messageCh      chan *fluent.FluentRecordSet
	monitorCh      chan Stat
	eventCh        chan fsnotify.Event
	format         FileFormat
	recordModifier *RecordModifier
	regexp         *Regexp
	position       int64
}

type Watcher struct {
	watcher      *fsnotify.Watcher
	watchingDir  map[string]bool
	watchingFile map[string]chan fsnotify.Event
}

func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("[error] Couldn't create file watcher", err)
		return nil, err
	}
	w := &Watcher{
		watcher:      watcher,
		watchingDir:  make(map[string]bool),
		watchingFile: make(map[string]chan fsnotify.Event),
	}
	return w, nil
}

func (w *Watcher) Run(c *Context) {
	c.InputProcess.Add(1)
	defer c.InputProcess.Done()
	c.StartProcess.Done()

	if len(w.watchingFile) == 0 {
		// no need to watch
		return
	}
	for {
		select {
		case <-c.ControlCh:
			log.Println("[info] shutdown file watcher")
			return
		case ev := <-w.watcher.Events:
			if eventCh, ok := w.watchingFile[ev.Name]; ok {
				eventCh <- ev
			}
		case err := <-w.watcher.Errors:
			log.Println("[warning] watcher error", err)
		}
	}
}

func (w *Watcher) WatchFile(filename string) (chan fsnotify.Event, error) {
	parent := filepath.Dir(filename)
	log.Println("[info] watching events of directory", parent)
	if _, ok := w.watchingDir[parent]; ok { // already watching
		ch := make(chan fsnotify.Event)
		w.watchingFile[filename] = ch
		return ch, nil
	} else {
		err := w.watcher.Add(parent)
		if err != nil {
			log.Println("[error] Couldn't watch event of", parent, err)
			return nil, err
		}
		w.watchingDir[parent] = true
		ch := make(chan fsnotify.Event)
		w.watchingFile[filename] = ch
		return ch, nil
	}
}

func Rel2Abs(filename string) (string, error) {
	if filepath.IsAbs(filename) {
		return filename, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Println("[error] Couldn't get current working dir.", err)
		return "", err
	}
	return filepath.Join(cwd, filename), nil
}

func NewInTail(config *ConfigLogfile, watcher *Watcher) (*InTail, error) {
	modifier := &RecordModifier{
		convertMap:    config.ConvertMap,
		timeParse:     config.TimeParse,
		timeKey:       config.TimeKey,
		timeConverter: TimeConverter(config.TimeFormat),
	}
	if config.IsStdin() {
		return &InTail{
			filename:       StdinFilename,
			tag:            config.Tag,
			fieldName:      config.FieldName,
			format:         config.Format,
			recordModifier: modifier,
		}, nil
	}

	filename, err := Rel2Abs(config.File)
	if err != nil {
		return nil, err
	}
	eventCh, err := watcher.WatchFile(filename)
	if err != nil {
		return nil, err
	}
	return &InTail{
		filename:       filename,
		tag:            config.Tag,
		fieldName:      config.FieldName,
		lastReadAt:     time.Now(),
		eventCh:        eventCh,
		format:         config.Format,
		recordModifier: modifier,
		regexp:         config.Regexp,
	}, nil
}

// InTail follow the tail of file and post BulkMessage to channel.
func (t *InTail) Run(c *Context) {
	c.InputProcess.Add(1)
	defer c.InputProcess.Done()

	t.messageCh = c.MessageCh
	t.monitorCh = c.MonitorCh

	c.StartProcess.Done()

	if t.eventCh == nil {
		err := t.TailStdin(c)
		if err != nil {
			if _, ok := err.(Signal); ok {
				log.Println("[info]", err)
			} else {
				log.Println("[error]", err)
			}
			return
		}
	}

	log.Println("[info] Trying trail file", t.filename)
	f, err := t.newTrailFile(SEEK_TAIL, c)
	if err != nil {
		if _, ok := err.(Signal); ok {
			log.Println("[info]", err)
		} else {
			log.Println("[error]", err)
		}
		return
	}
	for {
		for {
			err := t.watchFileEvent(f, c)
			if err != nil {
				if _, ok := err.(Signal); ok {
					log.Println("[info]", err)
					return
				} else {
					log.Println("[warning]", err)
					break
				}
			}
		}
		// re open file
		var err error
		f, err = t.newTrailFile(SEEK_HEAD, c)
		if err != nil {
			if _, ok := err.(Signal); ok {
				log.Println("[info]", err)
			} else {
				log.Println("[error]", err)
			}
			return
		}
	}
}

func (t *InTail) newTrailFile(startPos int64, c *Context) (*File, error) {
	seekTo := startPos
	first := true
	for {
		f, err := openFile(t.filename, seekTo)
		if err == nil {
			f.Tag = t.tag
			f.FieldName = t.fieldName
			f.Format = t.format
			f.RecordModifier = t.recordModifier
			f.Regexp = t.regexp
			log.Println("[info] Trailing file:", f.Path, "tag:", f.Tag, "format:", t.format)
			t.monitorCh <- f.UpdateStat()
			return f, nil
		}
		t.monitorCh <- &FileStat{
			Tag:      t.tag,
			File:     t.filename,
			Position: int64(-1),
			Error:    monitorError(err),
		}
		if first {
			log.Println("[warning]", err, "Retrying...")
		}
		first = false
		seekTo = SEEK_HEAD
		select {
		case <-c.ControlCh:
			return nil, Signal{"shutdown in_tail: " + t.filename}
		case <-time.NewTimer(OpenRetryInterval).C:
		}
	}
}

func (t *InTail) watchFileEvent(f *File, c *Context) error {
	select {
	case <-c.ControlCh:
		return Signal{"shutdown in_tail: " + f.Path}
	case ev := <-t.eventCh:
		if ev.Op&fsnotify.Write == fsnotify.Write {
			break
		}
		if ev.Op&fsnotify.Remove == fsnotify.Remove || ev.Op&fsnotify.Rename == fsnotify.Rename {
			log.Println("[info] fsevent", ev.String())
			f.tailAndSend(t.messageCh, t.monitorCh)
			f.Close()
			return errors.New(t.filename + " was closed")
		} else if ev.Op&fsnotify.Create == fsnotify.Create {
			return nil
		}
	case <-time.After(TailInterval):
	}

	err := f.restrict()
	if err != nil {
		return err
	}
	if time.Now().Before(t.lastReadAt.Add(TailInterval)) {
		return nil
	}
	err = f.tailAndSend(t.messageCh, t.monitorCh)
	t.lastReadAt = time.Now()

	if err != io.EOF {
		log.Println(err)
		return err
	}
	return nil
}

func (t *InTail) TailStdin(c *Context) error {
	t.monitorCh <- &FileStat{
		Tag:      t.tag,
		File:     t.filename,
		Position: 0,
	}
	go func() {
		<-c.ControlCh
		os.Stdin.Close()
	}()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		b := scanner.Bytes()
		t.position += int64(len(b) + 1)
		t.messageCh <- NewFluentRecordSet(t.tag, t.fieldName, t.format, t.recordModifier, t.regexp, b)
		t.monitorCh <- &FileStat{
			File:     StdinFilename,
			Position: t.position,
			Tag:      t.tag,
		}
	}
	var msg string
	if err := scanner.Err(); err != nil {
		msg = err.Error()
	} else {
		msg = "closed"
	}
	t.monitorCh <- &FileStat{
		File:     StdinFilename,
		Position: t.position,
		Tag:      t.tag,
		Error:    msg,
	}
	return NewSignal("shutdown in_tail: STDIN")
}
