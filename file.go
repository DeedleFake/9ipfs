package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DeedleFake/p9"
)

type file struct {
	fs   FileSystem
	path string
}

func (f file) ReadAt(p []byte, off int64) (n int, err error) {
	rsp, err := http.Get(f.fs.endpoint("cat", "arg", f.path, "offset", off, "length", len(p)))
	if err != nil {
		return 0, err
	}
	defer rsp.Body.Close()

	n, err = rsp.Body.Read(p)
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func (f file) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, errors.New("read-only filesystem")
}

func (f file) Close() error {
	return nil
}

func (f file) Readdir() ([]p9.DirEntry, error) {
	rsp, err := http.Get(f.fs.endpoint("ls", "arg", f.path))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	var data struct {
		Objects []struct {
			Links []struct {
				Name string
				Size uint64
				Type int32
			}
		}
	}
	err = json.NewDecoder(rsp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	if len(data.Objects) == 0 {
		return nil, nil
	}

	entries := make([]p9.DirEntry, 0, len(data.Objects[0].Links))
	for _, link := range data.Objects[0].Links {
		mode := p9.FileMode(0444)
		if link.Type == 1 {
			mode = p9.ModeDir | 0555
		}

		entries = append(entries, p9.DirEntry{
			Mode:   mode,
			Length: link.Size,
			Name:   link.Name,
		})
	}

	return entries, nil
}
