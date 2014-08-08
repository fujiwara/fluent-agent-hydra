package hydra

import (
	"bytes"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"io"
	"log"
	"os"
	"time"
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
	Path       string
	Tag        string
	Position   int64
	contBuf    []byte
	lastStat   os.FileInfo
	FieldName  string
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
			monitorCh <- &FileStat{
				Tag:      tag,
				File:     path,
				Position: f.Position,
			}
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
	for {
		readBuf := make([]byte, ReadBufferSize)
		sendBuf := make([]byte, 0, ReadBufferSize*2)
		n, err := io.ReadFull(f, readBuf)
		if n == 0 {
			return err
		}
		f.Position += int64(n)
		blockLen := bytes.LastIndex(readBuf, LineSeparator)
		if DEBUG {
			log.Println("read", n, "blockLen", blockLen)
		}
		if blockLen == -1 {
			// whole of readBuf is continuous line
			f.contBuf = append(f.contBuf, readBuf[0:n]...)
			continue
		} else if blockLen == n-1 {
			// readBuf is just terminated by '\n'
			sendBuf = append(sendBuf, f.contBuf...)
			sendBuf = append(sendBuf, readBuf[0:n-1]...)
			f.contBuf = []byte{}
		} else {
			// bottom line of readBuf is continuous line
			if DEBUG {
				log.Println("contBuf", f.contBuf)
			}
			sendBuf = append(sendBuf, f.contBuf...)
			sendBuf = append(sendBuf, readBuf[0:blockLen]...)
			f.contBuf = readBuf[blockLen+1 : n]
		}
		messageCh <- NewFluentRecordSet(f.Tag, f.FieldName, &sendBuf)
		monitorCh <- &FileStat{
			File:     f.Path,
			Position: f.Position,
			Tag:      f.Tag,
		}
	}
	return nil
}
