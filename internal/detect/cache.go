package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type cacheFile struct {
	Entries map[string]cacheEntry `json:"entries"`
}

type cacheEntry struct {
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Format string `json:"format"`
}

var (
	cacheMu   sync.Mutex
	cachePath = "" // overridable in tests via SetCachePath
)

// SetCachePath overrides the sniff cache file path (tests).
func SetCachePath(path string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cachePath = path
}

func resolveCachePath() (string, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cachePath != "" {
		return cachePath, nil
	}
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sniff.json"), nil
}

func cacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "zipstar"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "zipstar"), nil
}

func absPath(path string) (string, error) {
	return filepath.Abs(path)
}

// LookupCache returns a cached format name when the identity still matches.
func LookupCache(id FileIdentity) (format string, ok bool) {
	path, err := resolveCachePath()
	if err != nil {
		return "", false
	}
	cf, err := loadCache(path)
	if err != nil {
		return "", false
	}
	ent, ok := cf.Entries[id.Path]
	if !ok {
		return "", false
	}
	if ent.Size != id.Size || ent.Mtime != id.Mtime {
		return "", false
	}
	if ent.Format == "" {
		return "", false
	}
	return ent.Format, true
}

// StoreCache writes a format name for the given identity.
func StoreCache(id FileIdentity, format string) error {
	path, err := resolveCachePath()
	if err != nil {
		return err
	}
	cf, err := loadCache(path)
	if err != nil {
		cf = cacheFile{Entries: map[string]cacheEntry{}}
	}
	if cf.Entries == nil {
		cf.Entries = map[string]cacheEntry{}
	}
	cf.Entries[id.Path] = cacheEntry{
		Size:   id.Size,
		Mtime:  id.Mtime,
		Format: format,
	}
	return saveCache(path, cf)
}

// SniffFormatCached returns a format name using cache then sniff.
func SniffFormatCached(path string) (name string, ok bool, err error) {
	id, err := IdentityOf(path)
	if err != nil {
		return "", false, err
	}
	if cached, hit := LookupCache(id); hit {
		return cached, true, nil
	}
	name, ok, err = SniffFormat(id.Path)
	if err != nil || !ok {
		return name, ok, err
	}
	_ = StoreCache(id, name)
	return name, true, nil
}

func loadCache(path string) (cacheFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheFile{}, err
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return cacheFile{}, err
	}
	if cf.Entries == nil {
		cf.Entries = map[string]cacheEntry{}
	}
	return cf, nil
}

func saveCache(path string, cf cacheFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}