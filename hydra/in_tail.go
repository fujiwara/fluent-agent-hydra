package hydra

import (
	"errors"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/howeyc/fsnotify"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	TailInterval = 200 * time.Millisecond
)

type InTail struct {
	filename   string
	tag        string
	fieldName  string
	messageCh  chan *fluent.FluentRecordSet
	monitorCh  chan Stat
	watcher    *fsnotify.Watcher
	lastReadAt time.Time
}

func NewInTail(config *ConfigLogfile, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (*InTail, error) {
	filename := config.File
	if !filepath.IsAbs(filename) { // rel path to abs path
		cwd, err := os.Getwd()
		if err != nil {
			log.Println("[error] Couldn't get current working dir.", err)
			return nil, err
		}
		filename = filepath.Join(cwd, filename)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("[error] Couldn't create file watcher", err)
		return nil, err
	}
	parent := filepath.Dir(filename)
	log.Println("[info] watching events of directory", parent)
	err = watcher.Watch(parent)
	if err != nil {
		log.Println("[error] Couldn't watch event of", parent, err)
		return nil, err
	}

	t := &InTail{
		filename:   filename,
		tag:        config.Tag,
		fieldName:  config.FieldName,
		messageCh:  messageCh,
		monitorCh:  monitorCh,
		watcher:    watcher,
		lastReadAt: time.Now(),
	}
	return t, nil
}

// InTail follow the tail of file and post BulkMessage to channel.
func (t *InTail) Run() {
	defer log.Println("[error] Aborted to in_tail.run()")
	defer t.watcher.Close()

	log.Println("[info] Trying trail file", t.filename)
	f := newTrailFile(t.filename, t.tag, t.fieldName, SEEK_TAIL, t.monitorCh)
	for {
		for {
			err := t.watchFileEvent(f)
			if err != nil {
				log.Println("[warning]", err)
				break
			}
		}
		// re open file
		f = newTrailFile(t.filename, t.tag, t.fieldName, SEEK_HEAD, t.monitorCh)
	}
}

func (t *InTail) watchFileEvent(f *File) error {
	select {
	case ev := <-t.watcher.Event:
		if ev.Name != t.filename {
			return nil
		}
		if ev.IsModify() {
			break
		}
		if ev.IsDelete() || ev.IsRename() {
			log.Println("[info]", ev.Name)
			f.tailAndSend(t.messageCh, t.monitorCh)
			f.Close()
			return errors.New(t.filename + " was closed")
		} else if ev.IsCreate() {
			return nil
		}
	case err := <-t.watcher.Error:
		log.Println("[error]", err)
		return err
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
