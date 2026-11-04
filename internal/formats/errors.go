package formats

import "errors"

// ErrArchiveNotSupported is returned by Format.Archive when the backend
// cannot create archives (e.g. RAR is extract/list only).
var ErrArchiveNotSupported = errors.New("creating archives is not supported for this format")
