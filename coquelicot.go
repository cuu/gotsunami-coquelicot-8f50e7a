// Package coquelicot provides (chunked) file upload capability (with resume).
package coquelicot

type Storage struct {
	output    string
	host			string
	verbosity int
}

// FIXME: global for now
var makeThumbnail bool

func (s *Storage) StorageDir() string {
	return s.output
}

func (s *Storage) Host() string {
	return "http://"+s.host
}

func NewStorage(rootDir string,h string) *Storage {
	return &Storage{output: rootDir,host:h}
}
