// Package rar implements RAR extract and list.
//
// Prefer the bundled/system UnRAR CLI when available (see FindUnrar); otherwise
// fall back to pure-Go github.com/nwaples/rardecode/v2.
// Creating RAR archives is not supported.
package rar

import (
	"context"
	"fmt"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// Format is the RAR backend (extract / list only).
type Format struct{}

// New returns a RAR Format.
func New() *Format { return &Format{} }

func (f *Format) Name() string         { return "rar" }
func (f *Format) Extensions() []string { return []string{".rar"} }

// Archive is unsupported for RAR.
func (f *Format) Archive(ctx context.Context, opts formats.ArchiveOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("rar: %w", formats.ErrArchiveNotSupported)
}

func (f *Format) Unarchive(ctx context.Context, opts formats.UnarchiveOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if bin, ok := FindUnrar(); ok {
		return extractUnrar(ctx, bin, opts)
	}
	return unarchiveDecode(ctx, opts)
}

func (f *Format) List(ctx context.Context, opts formats.ListOpts) ([]formats.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if bin, ok := FindUnrar(); ok {
		return listUnrar(ctx, bin, opts)
	}
	return listDecode(ctx, opts)
}
