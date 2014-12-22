package hydra

import (
	"bytes"
	"io"
	"log"
	"os"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

const (
	OpenRetryInterval = 1 * time.Second
	SEEK_TAIL         = int64(-1)
	SEEK_HEAD         = int64(0)
	DEBUG             = false
)

var (
	ReadBufferSize = 64 * 1024
)

type File struct {
	*os.File
	Path      string
	Tag       string
	Position  int64
	contBuf   []byte
	lastStat  os.FileInfo
	FieldName string
}

func newTrailFile(path string, tag string, fieldName string, startPos int64, monitorCh chan Stat) *File {
	seekTo := startPos
	first := true
	for {
		f, err := openFile(path, seekTo)
		if err == nil {
			f.Tag = tag
			f.FieldName = fieldName
			log.Println("[info] Trailing file:", f.Path, "tag:", f.Tag)
			monitorCh <- f.NewStat()
			return f
		}
		monitorCh <- &FileStat{
			Tag:      tag,
			File:     path,
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

func openFile(path string, startPos int64) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	file := &File{f, path, "", startPos, make([]byte, 0), stat, ""}

	if startPos == SEEK_TAIL {
		// seek to end of file
		size := file.lastStat.Size()
		pos, _ := file.Seek(size, os.SEEK_SET)
		file.Position = pos
	} else {
		pos, _ := file.Seek(startPos, os.SEEK_SET)
		file.Position = pos
	}
	log.Println("[info]", file.Path, "Seeked to", file.Position)
	return file, nil
}

func (f *File) restrict() error {
	var err error
	f.lastStat, err = f.Stat()
	if err != nil {
		log.Println("[error]", f.Path, "stat failed", err)
		return err
	}
	if size := f.lastStat.Size(); size < f.Position {
		pos, _ := f.Seek(int64(0), os.SEEK_SET)
		f.Position = pos
		log.Println("[info]", f.Path, "was truncated. Seeked to", pos)
	}
	return nil
}

func (f *File) tailAndSend(messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) error {
	readBuf := make([]byte, ReadBufferSize)
	for {
		sendBuf := make([]byte, 0)
		n, err := io.ReadAtLeast(f, readBuf, 1)
		if n == 0 {
			return err
		}
		f.Position += int64(n)
		if readBuf[n-1] == '\n' {
			// readBuf is just terminated by '\n'
			if len(f.contBuf) > 0 {
				sendBuf = append(sendBuf, f.contBuf...)
				f.contBuf = []byte{}
			}
			sendBuf = append(sendBuf, readBuf[0:n-1]...)
		} else {
			blockLen := bytes.LastIndex(readBuf[0:n], LineSeparator)
			if blockLen == -1 {
				// whole of readBuf is continuous line
				f.contBuf = append(f.contBuf, readBuf[0:n]...)
				continue
			} else {
				// bottom line of readBuf is continuous line
				if len(f.contBuf) > 0 {
					sendBuf = append(sendBuf, f.contBuf...)
					f.contBuf = make([]byte, n-blockLen-1)
					copy(f.contBuf, readBuf[blockLen+1:n])
					log.Println("    newc", string(f.contBuf))
				}
				sendBuf = append(sendBuf, readBuf[0:blockLen]...)
			}
		}
		messageCh <- NewFluentRecordSet(f.Tag, f.FieldName, &sendBuf)
		monitorCh <- f.NewStat()
	}
}

func (f *File) NewStat() *FileStat {
	return &FileStat{
		File:     f.Path,
		Position: f.Position,
		Tag:      f.Tag,
	}
}
