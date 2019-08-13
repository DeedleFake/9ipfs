package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
