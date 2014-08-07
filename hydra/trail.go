package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/howeyc/fsnotify"
	"io"
	"log"
	"os"
	"path/filepath"
)

// InTail follow the tail of file and post BulkMessage to channel.
func InTail(conf ConfigLogfile, ch chan *fluent.FluentRecordSet) {
	filename := conf.File
	tag := conf.Tag
	fieldName := conf.FieldName
	defer log.Println("[error] Aborted to trail")

	if !filepath.IsAbs(filename) { // rel path to abs path
		cwd, err := os.Getwd()
		if err != nil {
			log.Println("[error] Couldn't get current working dir.", err)
			return
		}
		conf.File = filepath.Join(cwd, filename)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("[error] Couldn't create file watcher", err)
		return
	}
	defer watcher.Close()

	parent := filepath.Dir(filename)
	log.Println("[info] watching events of directory", parent)
	err = watcher.Watch(parent)
	if err != nil {
		log.Println("[error] Couldn't watch event of", parent, err)
		return
	}
	log.Println("[info] Trying trail file", filename)
	f := newTrailFile(filename, tag, fieldName, SEEK_TAIL)
	defer f.Close()

EVENT:
	for {
		select {
		case ev := <-watcher.Event:
			if ev.Name != filename {
				continue EVENT // ignore
			}
			if ev.IsDelete() || ev.IsRename() {
				log.Println("[info]", ev)
				f.tailAndSend(ch)
				f.Close()
				f = newTrailFile(filename, tag, fieldName, SEEK_HEAD)
			} else {
				f.restrict()
			}
			err = f.tailAndSend(ch)
			if err != io.EOF {
				log.Println(err)
			}
		case err := <-watcher.Error:
			log.Println("error:", err)
			watcher.RemoveWatch(parent)
			watcher.Watch(parent)
		}
	}
}
