package zip

import (
	"compress/flate"
	"io"

	stdzip "archive/zip"
)

// newStdFlateCompressor returns a per-Writer compressor for archive/zip at level 0-9.
func newStdFlateCompressor(level int) stdzip.Compressor {
	return func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, level)
	}
}
