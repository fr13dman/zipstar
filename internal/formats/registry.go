package formats

import (
	"fmt"
	"strings"
	"sync"
)

// Registry maps format names and file extensions to Format implementations.
type Registry struct {
	mu         sync.RWMutex
	byName     map[string]Format
	byExt      map[string]Format // key is lowercased extension including dot
}

// NewRegistry returns an empty format registry.
func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]Format),
		byExt:  make(map[string]Format),
	}
}

// Register adds a format. Panics on duplicate name or extension so miswiring
// fails at startup instead of during a user command.
func (r *Registry) Register(f Format) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(f.Name())
	if _, exists := r.byName[name]; exists {
		panic(fmt.Sprintf("formats: duplicate format name %q", name))
	}
	r.byName[name] = f

	for _, ext := range f.Extensions() {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			panic(fmt.Sprintf("formats: extension %q must start with a dot", ext))
		}
		if _, exists := r.byExt[ext]; exists {
			panic(fmt.Sprintf("formats: duplicate extension %q", ext))
		}
		r.byExt[ext] = f
	}
}

// Get returns a format by name (case-insensitive).
func (r *Registry) Get(name string) (Format, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.byName[strings.ToLower(name)]
	return f, ok
}

// ByExtension returns a format for an extension such as ".zip".
func (r *Registry) ByExtension(ext string) (Format, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.byExt[strings.ToLower(ext)]
	return f, ok
}

// Names returns registered format names in arbitrary order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byName))
	for name := range r.byName {
		out = append(out, name)
	}
	return out
}
