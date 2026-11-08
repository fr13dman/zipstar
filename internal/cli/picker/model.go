package picker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fr13dman/zipstar-cli/internal/archive"
)

// ErrCanceled is returned when the user quits the picker without a selection.
var ErrCanceled = fmt.Errorf("archive selection canceled")

type dirEntry struct {
	Name      string
	IsDir     bool
	IsArchive bool
	Parts     int // > 1 for the first volume of a multi-volume set (other volumes are hidden)
	ModTime   time.Time
}

type sortField int

const (
	sortByDate sortField = iota
	sortByName
)

// model is the Bubble Tea path picker state.
//
// dir is the committed directory (changed by Enter on a folder). viewDir is the
// directory currently listed: it follows a path typed into the input (e.g.
// "~/Downloads/" or "../other/x"), resolved relative to dir, so typing a path
// previews that folder before Enter commits it.
type model struct {
	dir       string
	viewDir   string
	exts      map[string]struct{}
	entries   []dirEntry // full listing of viewDir
	filtered  []dirEntry
	cursor    int
	filter    textinput.Model
	selected  string
	canceled  bool
	statusErr string
	quitting  bool
	sortBy    sortField
	sortDesc  bool // date: newest first; name: Z-A
}

func newModel(startDir string, extensions []string) model {
	exts := make(map[string]struct{}, len(extensions))
	for _, e := range extensions {
		exts[strings.ToLower(e)] = struct{}{}
	}
	ti := textinput.New()
	ti.Placeholder = "filter or path…"
	ti.Focus()
	ti.CharLimit = 2048
	ti.Width = 48

	m := model{
		dir:      startDir,
		viewDir:  startDir,
		exts:     exts,
		filter:   ti,
		sortBy:   sortByDate,
		sortDesc: true,
	}
	m.reload()
	return m
}

func (m *model) reload() {
	m.statusErr = ""
	ents, err := os.ReadDir(m.viewDir)
	if err != nil {
		m.entries = nil
		m.filtered = nil
		if os.IsNotExist(err) {
			m.statusErr = "no such directory: " + m.viewDir
		} else {
			m.statusErr = err.Error()
		}
		return
	}
	out := make([]dirEntry, 0, len(ents)+1)
	parent := filepath.Dir(m.viewDir)
	if parent != m.viewDir {
		out = append(out, dirEntry{Name: "..", IsDir: true})
	}
	sets := volumeSets(ents)
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		isDir := e.IsDir()
		isArch := false
		parts := 0
		if !isDir {
			ext := strings.ToLower(filepath.Ext(name))
			_, isArch = m.exts[ext]
			if key, _, ok := archive.VolumeOf(name); ok {
				if set := sets[key]; set.count > 1 {
					if name != set.first {
						continue // later volumes are reached through the first one
					}
					isArch = true
					parts = set.count
				}
			}
		}
		var mod time.Time
		if info, err := e.Info(); err == nil {
			mod = info.ModTime()
		}
		out = append(out, dirEntry{Name: name, IsDir: isDir, IsArchive: isArch, Parts: parts, ModTime: mod})
	}
	m.entries = out
	m.sortEntries()
}

// sortEntries orders entries by the current sort, keeping ".." on top, and
// refreshes the filtered view.
func (m *model) sortEntries() {
	rest := m.entries
	if len(rest) > 0 && rest[0].Name == ".." {
		rest = rest[1:]
	}
	sort.SliceStable(rest, func(i, j int) bool {
		return m.entryLess(rest[i], rest[j])
	})
	m.applyFilter()
}

func (m *model) entryLess(a, b dirEntry) bool {
	if m.sortBy == sortByDate && !a.ModTime.Equal(b.ModTime) {
		if m.sortDesc {
			return a.ModTime.After(b.ModTime)
		}
		return a.ModTime.Before(b.ModTime)
	}
	if m.sortBy == sortByName && m.sortDesc {
		return nameLess(b.Name, a.Name)
	}
	return nameLess(a.Name, b.Name)
}

func nameLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a < b
}

// setSort re-sorts the listing and keeps the highlighted entry highlighted.
func (m *model) setSort(field sortField, desc bool) {
	cur := ""
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		cur = m.filtered[m.cursor].Name
	}
	m.sortBy, m.sortDesc = field, desc
	m.sortEntries()
	for i, e := range m.filtered {
		if e.Name == cur {
			m.cursor = i
			return
		}
	}
}

func (m model) sortLabel() string {
	switch {
	case m.sortBy == sortByName && m.sortDesc:
		return "sort: name (Z-A)"
	case m.sortBy == sortByName:
		return "sort: name (A-Z)"
	case m.sortDesc:
		return "sort: date (newest first)"
	default:
		return "sort: date (oldest first)"
	}
}

type volumeSet struct {
	first    string
	firstNum int
	count    int
}

// volumeSets groups multi-volume RAR names in a listing by set, recording each
// set's first volume (the one extraction must start from) and size.
func volumeSets(ents []os.DirEntry) map[string]volumeSet {
	sets := map[string]volumeSet{}
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		key, num, ok := archive.VolumeOf(e.Name())
		if !ok {
			continue
		}
		s := sets[key]
		if s.count == 0 || num < s.firstNum {
			s.first, s.firstNum = e.Name(), num
		}
		s.count++
		sets[key] = s
	}
	return sets
}

func (m *model) applyFilter() {
	_, q, _ := m.inputView()
	q = strings.ToLower(q)
	if q == "" {
		m.filtered = append([]dirEntry(nil), m.entries...)
	} else {
		m.filtered = m.filtered[:0]
		for _, e := range m.entries {
			if strings.Contains(strings.ToLower(e.Name), q) {
				m.filtered = append(m.filtered, e)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// inputView interprets the input as either a plain filter for dir, or a path:
// the part up to the last separator picks the directory to list and the rest
// filters inside it.
func (m *model) inputView() (viewDir, query string, isPath bool) {
	raw := strings.TrimSpace(m.filter.Value())
	if raw == "" || !looksLikePath(raw) {
		return m.dir, raw, false
	}
	dirRaw, q := splitPathInput(raw)
	if dirRaw == "" {
		return m.dir, q, true
	}
	return filepath.Clean(expandPath(dirRaw, m.dir)), q, true
}

// onInputChanged re-lists when the typed path points at another directory,
// then filters the listing by the remainder.
func (m *model) onInputChanged() {
	m.cursor = 0
	m.statusErr = ""
	viewDir, _, _ := m.inputView()
	if viewDir != m.viewDir || m.entries == nil {
		m.viewDir = viewDir
		m.reload()
		return
	}
	m.applyFilter()
}

// splitPathInput splits "a/b/fil" into ("a/b/", "fil"). A trailing "~", "." or
// ".." component names a directory, so it stays in the directory part.
func splitPathInput(raw string) (dirRaw, query string) {
	dirRaw, query = "", raw
	if i := strings.LastIndexAny(raw, "/"+string(filepath.Separator)); i >= 0 {
		dirRaw, query = raw[:i+1], raw[i+1:]
	}
	if query == "~" || query == "." || query == ".." {
		dirRaw, query = raw, ""
	}
	return dirRaw, query
}

// enterDir commits dir as the current directory and clears the input.
func (m *model) enterDir(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		m.statusErr = err.Error()
		return
	}
	m.dir = abs
	m.viewDir = abs
	m.filter.SetValue("")
	m.cursor = 0
	m.reload()
}

// complete replaces the input with the highlighted entry's path (Tab).
// Directories get a trailing separator so the listing previews their contents.
func (m *model) complete() {
	if len(m.filtered) == 0 {
		return
	}
	cur := m.filtered[m.cursor]
	prefix := ""
	if _, _, isPath := m.inputView(); isPath {
		dirRaw, _ := splitPathInput(strings.TrimSpace(m.filter.Value()))
		prefix = dirRaw
		if prefix != "" && !strings.HasSuffix(prefix, "/") && !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
	}
	val := prefix + cur.Name
	if cur.IsDir {
		val += string(filepath.Separator)
	}
	m.filter.SetValue(val)
	m.filter.CursorEnd()
	m.onInputChanged()
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			m.quitting = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			return m.handleEnter()
		case tea.KeyTab:
			m.complete()
			return m, nil
		case tea.KeyCtrlS:
			if m.sortBy == sortByDate {
				m.setSort(sortByName, false)
			} else {
				m.setSort(sortByDate, true)
			}
			return m, nil
		case tea.KeyCtrlR:
			m.setSort(m.sortBy, !m.sortDesc)
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.filter.Value()
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prev {
		m.onInputChanged()
	}
	return m, cmd
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.filter.Value())
	// An exact typed path wins: open a directory, select a file.
	if raw != "" && looksLikePath(raw) {
		target := expandPath(raw, m.dir)
		if info, err := os.Stat(target); err == nil {
			if info.IsDir() {
				m.enterDir(target)
				return m, nil
			}
			abs, err := filepath.Abs(target)
			if err != nil {
				m.statusErr = err.Error()
				return m, nil
			}
			m.selected = abs
			m.quitting = true
			return m, tea.Quit
		}
		// Partial path (e.g. "~/Downloads/fo"): fall through to the highlighted
		// match inside the previewed directory.
	}

	if len(m.filtered) == 0 {
		if m.statusErr == "" {
			m.statusErr = "no matching entries"
		}
		return m, nil
	}
	cur := m.filtered[m.cursor]
	full := filepath.Join(m.viewDir, cur.Name)
	if cur.Name == ".." {
		full = filepath.Dir(m.viewDir)
	}
	if cur.IsDir {
		m.enterDir(full)
		return m, nil
	}
	if !cur.IsArchive {
		m.statusErr = "select an archive file (or type a path)"
		return m, nil
	}
	abs, err := filepath.Abs(full)
	if err != nil {
		m.statusErr = err.Error()
		return m, nil
	}
	m.selected = abs
	m.quitting = true
	return m, tea.Quit
}

func looksLikePath(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") || strings.HasPrefix(s, ".") {
		return true
	}
	return strings.Contains(s, string(filepath.Separator))
}

func expandPath(s, base string) string {
	if strings.HasPrefix(s, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			s = filepath.Join(home, strings.TrimPrefix(s, "~"))
		}
	}
	if filepath.IsAbs(s) {
		return s
	}
	return filepath.Join(base, s)
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	archStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dirStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select archive"))
	b.WriteString("\n")
	b.WriteString(m.viewDir)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.sortLabel()))
	b.WriteString("\n\n")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	const maxRows = 16
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := start; i < end; i++ {
		e := m.filtered[i]
		label := e.Name
		if e.IsDir {
			label = dirStyle.Render(e.Name + "/")
		} else if e.IsArchive {
			label = archStyle.Render(e.Name)
			if e.Parts > 1 {
				label += helpStyle.Render(fmt.Sprintf(" (%d parts)", e.Parts))
			}
		}
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}
		b.WriteString(prefix)
		b.WriteString(label)
		b.WriteString("\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString(helpStyle.Render("(empty)"))
		b.WriteString("\n")
	}
	if m.statusErr != "" {
		b.WriteString("\n")
		b.WriteString(errStyle.Render(m.statusErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("type to filter or a path (/, ~/, ../) to browse · tab complete · ↑↓ move · enter open/select · ctrl+s sort name/date · ctrl+r reverse · esc cancel"))
	b.WriteString("\n")
	return b.String()
}
