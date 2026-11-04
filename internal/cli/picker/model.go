package picker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCanceled is returned when the user quits the picker without a selection.
var ErrCanceled = fmt.Errorf("archive selection canceled")

type dirEntry struct {
	Name      string
	IsDir     bool
	IsArchive bool
}

// model is the Bubble Tea path picker state.
type model struct {
	dir        string
	exts       map[string]struct{}
	entries    []dirEntry // full listing of current dir
	filtered   []dirEntry
	cursor     int
	filter     textinput.Model
	selected   string
	canceled   bool
	statusErr  string
	quitting   bool
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
		dir:    startDir,
		exts:   exts,
		filter: ti,
	}
	m.reload()
	return m
}

func (m *model) reload() {
	m.statusErr = ""
	ents, err := os.ReadDir(m.dir)
	if err != nil {
		m.entries = nil
		m.filtered = nil
		m.statusErr = err.Error()
		return
	}
	out := make([]dirEntry, 0, len(ents)+1)
	parent := filepath.Dir(m.dir)
	if parent != m.dir {
		out = append(out, dirEntry{Name: "..", IsDir: true})
	}
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		isDir := e.IsDir()
		isArch := false
		if !isDir {
			ext := strings.ToLower(filepath.Ext(name))
			_, isArch = m.exts[ext]
		}
		out = append(out, dirEntry{Name: name, IsDir: isDir, IsArchive: isArch})
	}
	m.entries = out
	m.applyFilter()
}

func (m *model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
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
		}
	}

	var cmd tea.Cmd
	prev := m.filter.Value()
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prev {
		m.applyFilter()
	}
	return m, cmd
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.filter.Value())
	// Jump/select when the filter looks like an explicit path.
	if raw != "" && looksLikePath(raw) {
		target := expandPath(raw, m.dir)
		info, err := os.Stat(target)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		if info.IsDir() {
			abs, err := filepath.Abs(target)
			if err != nil {
				m.statusErr = err.Error()
				return m, nil
			}
			m.dir = abs
			m.filter.SetValue("")
			m.cursor = 0
			m.reload()
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

	if len(m.filtered) == 0 {
		m.statusErr = "no matching entries"
		return m, nil
	}
	cur := m.filtered[m.cursor]
	full := filepath.Join(m.dir, cur.Name)
	if cur.Name == ".." {
		full = filepath.Dir(m.dir)
	}
	if cur.IsDir {
		abs, err := filepath.Abs(full)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.dir = abs
		m.filter.SetValue("")
		m.cursor = 0
		m.reload()
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
	b.WriteString(m.dir)
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
	b.WriteString(helpStyle.Render("↑↓ move · enter open/select · esc cancel"))
	b.WriteString("\n")
	return b.String()
}
