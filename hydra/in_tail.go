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

func (w *Watcher) Run() {
	InputProcessGroup.Add(1)
	defer InputProcessGroup.Done()

	if len(w.watchingFile) == 0 {
		log.Println("[error] no watching file. watcher aborted.")
		return
	}
	for {
		select {
		case <-ControlCh:
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

func NewInTail(config *ConfigLogfile, watcher *Watcher, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (*InTail, error) {
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
			messageCh:      messageCh,
			monitorCh:      monitorCh,
			format:         config.Format,
			recordModifier: modifier,
		}, nil
	} else {
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
			messageCh:      messageCh,
			monitorCh:      monitorCh,
			eventCh:        eventCh,
			format:         config.Format,
			recordModifier: modifier,
			regexp:         config.Regexp,
		}, nil
	}
}

// InTail follow the tail of file and post BulkMessage to channel.
func (t *InTail) Run() {
	InputProcessGroup.Add(1)
	defer InputProcessGroup.Done()

	if t.eventCh == nil {
		t.TailStdin()
		return
	}

	log.Println("[info] Trying trail file", t.filename)
	f := t.newTrailFile(SEEK_TAIL)
	for {
		for {
			err := t.watchFileEvent(f)
			if err != nil {
				if _, ok := err.(*ShutdownType); ok {
					log.Println("[info]", err)
					return
				} else {
					log.Println("[warning]", err)
					break
				}
			}
		}
		// re open file
		f = t.newTrailFile(SEEK_HEAD)
	}
}

func (t *InTail) newTrailFile(startPos int64) *File {
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
			return f
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
		time.Sleep(OpenRetryInterval)
	}
}

func (t *InTail) watchFileEvent(f *File) error {
	select {
	case <-ControlCh:
		return &ShutdownType{"shutdown in_tail: " + f.Path}
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

func (t *InTail) TailStdin() {
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
	if err := scanner.Err(); err != nil {
		log.Println("reading stdin:", err)
		t.monitorCh <- &FileStat{
			File:     StdinFilename,
			Position: t.position,
			Tag:      t.tag,
			Error:    err.Error(),
		}
	}
}
