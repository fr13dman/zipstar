// Package detect provides archive content sniffing and a small on-disk cache.
package detect

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Known zip signatures.
var (
	zipLocalFile = []byte{'P', 'K', 0x03, 0x04}
	zipEOCD      = []byte{'P', 'K', 0x05, 0x06}
	zipDataDesc  = []byte{'P', 'K', 0x07, 0x08}
	zipCentral   = []byte{'P', 'K', 0x01, 0x02}
)

// SniffFormat inspects file contents and returns a registered-style format name
// (e.g. "zip") when recognized. ok is false when the file is not a known archive.
func SniffFormat(path string) (name string, ok bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	head := make([]byte, 4)
	n, err := io.ReadFull(f, head)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		if n == 4 && looksLikeZipMagic(head) {
			return "zip", true, nil
		}
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if looksLikeZipMagic(head) {
		return "zip", true, nil
	}

	// Empty / nearly-empty zips may only have an EOCD near the end.
	info, err := f.Stat()
	if err != nil {
		return "", false, err
	}
	size := info.Size()
	if size < 22 {
		return "", false, nil
	}
	window := int64(65536)
	if size < window {
		window = size
	}
	buf := make([]byte, window)
	if _, err := f.ReadAt(buf, size-window); err != nil && err != io.EOF {
		return "", false, err
	}
	if bytes.Contains(buf, zipEOCD) {
		return "zip", true, nil
	}
	return "", false, nil
}

func looksLikeZipMagic(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	return bytes.Equal(b[:4], zipLocalFile) ||
		bytes.Equal(b[:4], zipEOCD) ||
		bytes.Equal(b[:4], zipDataDesc) ||
		bytes.Equal(b[:4], zipCentral)
}

// FileIdentity is the cache key material for a path.
type FileIdentity struct {
	Path  string
	Size  int64
	Mtime int64 // UnixNano
}

// IdentityOf stats path for cache keys.
func IdentityOf(path string) (FileIdentity, error) {
	abs, err := absPath(path)
	if err != nil {
		return FileIdentity{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return FileIdentity{}, err
	}
	if info.IsDir() {
		return FileIdentity{}, fmt.Errorf("path is a directory: %s", abs)
	}
	return FileIdentity{
		Path:  abs,
		Size:  info.Size(),
		Mtime: info.ModTime().UnixNano(),
	}, nil
}
