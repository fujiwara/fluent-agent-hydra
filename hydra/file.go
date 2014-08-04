package hydra

import (
	"bytes"
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
	LineSeparator  = []byte{'\n'}
	ReadBufferSize = 64 * 1024
)

type File struct {
	*os.File
	Path     string
	Tag      string
	Position int64
	contBuf  []byte
	lastStat os.FileInfo
}

func newTrailFile(path string, tag string, startPos int64) *File {
	log.Println("attempt to open file", path)
	seekTo := startPos
	for {
		f, err := openFile(path, seekTo)
		if err == nil {
			f.Tag = tag
			log.Println("trailing file:", f.Path, "tag:", f.Tag)
			return f
		}
		seekTo = SEEK_HEAD
		log.Println("trying to open", path)
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

	file := &File{f, path, "", startPos, make([]byte, 0), stat}

	if startPos == SEEK_TAIL {
		// seek to end of file
		size := file.lastStat.Size()
		pos, _ := file.Seek(size, os.SEEK_SET)
		file.Position = pos
	} else {
		pos, _ := file.Seek(startPos, os.SEEK_SET)
		file.Position = pos
	}
	log.Println(file.Path, "Seeked position", file.Position)
	return file, nil
}

func (f *File) restrict() error {
	var err error
	f.lastStat, err = f.Stat()
	if err != nil {
		log.Println("file stat failed", err)
		return err
	}
	if size := f.lastStat.Size(); size < f.Position {
		pos, _ := f.Seek(int64(0), os.SEEK_SET)
		f.Position = pos
		log.Println(f.Path, "was truncated. seeked to", pos)
	}
	return nil
}

func (f *File) tailAndSend(ch chan *BulkMessage) error {
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
		ch <- NewBulkMessage(f.Tag, &sendBuf)
	}
	return nil
}
