package hydra

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/howeyc/fsnotify"
)

const (
	TailInterval = 200 * time.Millisecond
)

type InTail struct {
	filename   string
	tag        string
	fieldName  string
	lastReadAt time.Time
	messageCh  chan *fluent.FluentRecordSet
	monitorCh  chan Stat
	eventCh    chan *fsnotify.FileEvent
	format     FileFormat
}

type Watcher struct {
	watcher      *fsnotify.Watcher
	watchingDir  map[string]bool
	watchingFile map[string]chan *fsnotify.FileEvent
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
		watchingFile: make(map[string]chan *fsnotify.FileEvent),
	}
	return w, nil
}

func (w *Watcher) Run() {
	if len(w.watchingFile) == 0 {
		log.Println("[error] no watching file. watcher aborted.")
		return
	}
	for {
		select {
		case ev := <-w.watcher.Event:
			if eventCh, ok := w.watchingFile[ev.Name]; ok {
				eventCh <- ev
			}
		case err := <-w.watcher.Error:
			log.Println("[warning] watcher error", err)
		}
	}
}

func (w *Watcher) WatchFile(filename string) (chan *fsnotify.FileEvent, error) {
	parent := filepath.Dir(filename)
	log.Println("[info] watching events of directory", parent)
	if _, ok := w.watchingDir[parent]; ok { // already watching
		ch := make(chan *fsnotify.FileEvent)
		w.watchingFile[filename] = ch
		return ch, nil
	} else {
		err := w.watcher.Watch(parent)
		if err != nil {
			log.Println("[error] Couldn't watch event of", parent, err)
			return nil, err
		}
		w.watchingDir[parent] = true
		ch := make(chan *fsnotify.FileEvent)
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
	filename, err := Rel2Abs(config.File)
	if err != nil {
		return nil, err
	}
	eventCh, err := watcher.WatchFile(filename)
	if err != nil {
		return nil, err
	}

	t := &InTail{
		filename:   filename,
		tag:        config.Tag,
		fieldName:  config.FieldName,
		lastReadAt: time.Now(),
		messageCh:  messageCh,
		monitorCh:  monitorCh,
		eventCh:    eventCh,
		format:     config.Format,
	}
	return t, nil
}

// InTail follow the tail of file and post BulkMessage to channel.
func (t *InTail) Run() {
	defer log.Println("[error] Aborted to in_tail.run()")

	log.Println("[info] Trying trail file", t.filename)
	f := newTrailFile(t.filename, t.tag, t.fieldName, SEEK_TAIL, t.monitorCh, t.format)
	for {
		for {
			err := t.watchFileEvent(f)
			if err != nil {
				log.Println("[warning]", err)
				break
			}
		}
		// re open file
		f = newTrailFile(t.filename, t.tag, t.fieldName, SEEK_HEAD, t.monitorCh, t.format)
	}
}

func (t *InTail) watchFileEvent(f *File) error {
	select {
	case ev := <-t.eventCh:
		if ev.IsModify() {
			break
		}
		if ev.IsDelete() || ev.IsRename() {
			log.Println("[info] fsevent", ev)
			f.tailAndSend(t.messageCh, t.monitorCh)
			f.Close()
			return errors.New(t.filename + " was closed")
		} else if ev.IsCreate() {
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
