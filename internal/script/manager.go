package script

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Ploos-AS/Engo/internal/bot"
)

type Manager struct {
	dir string
	bot *bot.Bot
	mu sync.RWMutex
	runtimes map[string]*Runtime
}

func NewManager(dir string, b *bot.Bot) *Manager {
	return &Manager{dir: dir, bot: b, runtimes: make(map[string]*Runtime)}
}

// ReloadAll validates every .tengo script before making the new set active.
// If any script fails, the currently active set remains untouched.
func (m *Manager) ReloadAll() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil { return fmt.Errorf("read scripts directory: %w", err) }

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tengo") { continue }
		paths = append(paths, filepath.Join(m.dir, entry.Name()))
	}
	sort.Strings(paths)
	if len(paths) == 0 { return fmt.Errorf("no .tengo scripts in %s", m.dir) }

	reg := bot.NewRegistry()
	next := make(map[string]*Runtime, len(paths))
	for _, path := range paths {
		rt := New(path, m.bot)
		src, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("%s: %w", filepath.Base(path), err) }
		if err := rt.prepare(src, &reg); err != nil { return fmt.Errorf("%s: %w", filepath.Base(path), err) }
		rt.src = append([]byte(nil), src...)
		next[path] = rt
	}

	m.bot.Replace(reg)
	m.mu.Lock()
	m.runtimes = next
	m.mu.Unlock()
	return nil
}

func (m *Manager) Scripts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.runtimes))
	for path := range m.runtimes { out = append(out, filepath.Base(path)) }
	sort.Strings(out)
	return out
}
