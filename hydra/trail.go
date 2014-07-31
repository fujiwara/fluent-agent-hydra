package hydra

import (
	"github.com/howeyc/fsnotify"
	"io"
	"log"
	"path/filepath"
)

// Trail follow the tail of file and post BulkMessage to channel.
func Trail(filename string, tag string, ch chan *BulkMessage) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("[FATAL] Couldn't create file watcher", err)
		return
	}
	defer watcher.Close()

	parent := filepath.Dir(filename)
	log.Println("watching event for", parent)
	err = watcher.Watch(parent)
	if err != nil {
		log.Println("[FATAL] Couldn't watch event for", parent, err)
		return
	}
	f := newTrailFile(filename, tag, SEEK_TAIL)
	defer f.Close()

EVENT:
	for {
		select {
		case ev := <-watcher.Event:
			if ev.Name != filename {
				continue EVENT // ignore
			}
			if ev.IsDelete() || ev.IsRename() {
				log.Println(ev)
				f.tailAndSend(ch)
				f.Close()
				f = newTrailFile(filename, tag, SEEK_HEAD)
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
