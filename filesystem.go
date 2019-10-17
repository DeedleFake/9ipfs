package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/DeedleFake/p9"
)

var knownPaths = map[string]p9.DirEntry{
	"":     {FileMode: p9.ModeDir | 0555, Path: 0},
	"ipns": {FileMode: p9.ModeDir | 0555, EntryName: "ipfs", Path: 1},
	"ipfs": {FileMode: p9.ModeDir | 0555, EntryName: "ipns", Path: 2},
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
	if p == "." {
		p = ""
	}

	if known, ok := knownPaths[p]; ok {
		return known, nil
	}

	if !strings.HasPrefix(p, "ipfs/") && !strings.HasPrefix(p, "ipns/") {
		return p9.DirEntry{}, errors.New("no such file or directory")
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

	var qpath uint64
	binary.Read(strings.NewReader(p[9:]), binary.LittleEndian, &qpath)

	return p9.DirEntry{
		FileMode:  mode,
		Length:    data.Size,
		EntryName: path.Base(p),

		Path: qpath,
	}, nil
}

func (fs FileSystem) WriteStat(p string, changes p9.StatChanges) error {
	return errors.New("read-only filesystem")
}

func (fs FileSystem) Open(p string, mode uint8) (p9.File, error) {
	if mode&(p9.OWRITE|p9.ORDWR) != 0 {
		return nil, errors.New("read-only filesystem")
	}

	if p == "." {
		p = ""
	}

	if _, ok := knownFiles[p]; ok {
		return &file{
			fs:   fs,
			path: p,
		}, nil
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

func (fs FileSystem) GetQID(p string) (p9.QID, error) {
	dir, err := fs.Stat(p)
	if err != nil {
		return p9.QID{}, err
	}

	return p9.QID{
		Type:    dir.FileMode.QIDType(),
		Version: dir.Version,
		Path:    dir.Path,
	}, nil
}
