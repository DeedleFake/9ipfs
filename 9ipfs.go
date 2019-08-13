package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/DeedleFake/p9"
)

var knownPaths = map[string]p9.DirEntry{
	"":     {Mode: p9.ModeDir | 0555},
	"ipns": {Mode: p9.ModeDir | 0555},
	"ipfs": {Mode: p9.ModeDir | 0555},
}

type FileSystem struct {
	addr *url.URL
}

func newFS(addr string) (*FileSystem, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	return &FileSystem{addr: u}, nil
}

func (fs FileSystem) endpoint(p string, args ...interface{}) string {
	u := *fs.addr

	u.Path = path.Join(u.Path, p)

	q := u.Query()
	for i := 0; i < len(args); i += 2 {
		q.Add(fmt.Sprint(args[i]), fmt.Sprint(args[i+1]))
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (fs FileSystem) resolve(p string) (string, error) {
	rsp, err := http.Get(fs.endpoint("resolve", "arg", "/"+p))
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	var data struct {
		Path string
	}
	err = json.NewDecoder(rsp.Body).Decode(&data)
	return data.Path, err
}

func (fs FileSystem) Auth(user, aname string) (p9.File, error) {
	return nil, errors.New("auth not supported")
}

func (fs *FileSystem) Attach(afile p9.File, user, aname string) (p9.Attachment, error) {
	if aname != "" {
		return nil, fmt.Errorf("invalid attachment: %q", aname)
	}

	return fs, nil
}

func (fs FileSystem) Stat(p string) (p9.DirEntry, error) {
	if known, ok := knownPaths[p]; ok {
		return known, nil
	}

	p, err := fs.resolve(p)
	if err != nil {
		return p9.DirEntry{}, err
	}

	rsp, err := http.Get(fs.endpoint("files/stat", "arg", p))
	if err != nil {
		return p9.DirEntry{}, err
	}
	defer rsp.Body.Close()

	var data struct {
		Size uint64
		Type string
	}
	err = json.NewDecoder(rsp.Body).Decode(&data)
	if err != nil {
		return p9.DirEntry{}, err
	}

	mode := p9.FileMode(0444)
	if data.Type == "directory" {
		mode |= p9.ModeDir
	}

	return p9.DirEntry{
		Mode:   mode,
		Length: data.Size,
		Name:   path.Base(p),
	}, nil
}

func (fs FileSystem) WriteStat(p string, changes p9.StatChanges) error {
	return errors.New("read-only filesystem")
}

func (fs FileSystem) Open(p string, mode uint8) (p9.File, error) {
	if mode&(p9.OWRITE|p9.ORDWR) != 0 {
		return nil, errors.New("read-only filesystem")
	}

	p, err := fs.resolve(p)
	if err != nil {
		return nil, err
	}

	return &file{
		fs:   fs,
		path: p,
	}, nil
}

func (fs FileSystem) Create(p string, perm p9.FileMode, mode uint8) (p9.File, error) {
	return nil, errors.New("read-only filesystem")
}

func (fs FileSystem) Remove(p string) error {
	return errors.New("read-only filesystem")
}

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

func main() {
	fs, err := newFS("http://localhost:5001/api/v0")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = p9.ListenAndServe("unix", "ipfs.sock", p9.FSConnHandler(fs, 4096))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
