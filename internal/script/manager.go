package script

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ploos-AS/Engo/internal/bot"
)

type Manager struct {
	dir                string
	bot                *bot.Bot
	maxAllocs          int64
	stateDir           string
	httpHosts          []string
	httpTimeout        time.Duration
	httpMaxBody        int64
	capabilities       map[string]Capabilities
	permissions        map[string][]string
	commandPermissions map[string]string
	mu                 sync.RWMutex
	reloadMu           sync.Mutex
	runtimes           map[string]*Runtime
	disabled           map[string]bool
}

func NewManager(dir string, b *bot.Bot) *Manager { return NewManagerLimited(dir, b, 100000) }
func NewManagerLimited(dir string, b *bot.Bot, maxAllocs int64) *Manager {
	return NewManagerWithState(dir, b, maxAllocs, "")
}
func NewManagerWithCapabilities(dir string, b *bot.Bot, maxAllocs int64, stateDir string, httpHosts []string, httpTimeout time.Duration, httpMaxBody int64) *Manager {
	return &Manager{dir: dir, bot: b, maxAllocs: maxAllocs, stateDir: stateDir, httpHosts: httpHosts, httpTimeout: httpTimeout, httpMaxBody: httpMaxBody, capabilities: make(map[string]Capabilities), permissions: make(map[string][]string), commandPermissions: make(map[string]string), runtimes: make(map[string]*Runtime), disabled: make(map[string]bool)}
}
func NewManagerWithState(dir string, b *bot.Bot, maxAllocs int64, stateDir string) *Manager {
	return &Manager{dir: dir, bot: b, maxAllocs: maxAllocs, stateDir: stateDir, httpTimeout: 10 * time.Second, httpMaxBody: 262144, capabilities: make(map[string]Capabilities), runtimes: make(map[string]*Runtime), disabled: make(map[string]bool)}
}

func (m *Manager) SetCommandPermissions(p map[string]string) {
	snapshot := copyCommandPermissions(p)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandPermissions = snapshot
}
func (m *Manager) SetPermissions(p map[string][]string) {
	snapshot := copyPermissions(p)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permissions = snapshot
}
func (m *Manager) SetScriptCapabilities(name string, c Capabilities) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capabilities[name] = c
	return nil
}
func (m *Manager) scriptCapabilities(path string) Capabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities[filepath.Base(path)]
}

func (m *Manager) ReloadAll() error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return fmt.Errorf("read scripts directory: %w", err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tengo") || m.isDisabled(e.Name()) {
			continue
		}
		paths = append(paths, filepath.Join(m.dir, e.Name()))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("no enabled .tengo scripts in %s", m.dir)
	}
	return m.activate(paths)
}

func (m *Manager) Enable(name string) error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	path := filepath.Join(m.dir, name)
	if _, err := os.Stat(path); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.disabled, name)
	m.mu.Unlock()
	if err := m.reloadCurrent(); err != nil {
		m.mu.Lock()
		m.disabled[name] = true
		m.mu.Unlock()
		return err
	}
	return nil
}

func (m *Manager) Disable(name string) error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	path := filepath.Join(m.dir, name)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is not a script", name)
	}
	m.mu.Lock()
	m.disabled[name] = true
	m.mu.Unlock()
	if err := m.reloadCurrent(); err != nil {
		m.mu.Lock()
		delete(m.disabled, name)
		m.mu.Unlock()
		return err
	}
	return nil
}

func (m *Manager) Reload(name string) error {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	if m.isDisabled(name) {
		return fmt.Errorf("%s is disabled", name)
	}
	path := filepath.Join(m.dir, name)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is not a script", name)
	}
	// Registries are rebuilt atomically, so a named reload validates the target then
	// rebuilds the complete enabled script set. This preserves cross-script command/event
	// registrations and rollback semantics if any script fails validation.
	return m.reloadCurrent()
}

func (m *Manager) reloadCurrent() error {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tengo") || m.isDisabled(e.Name()) {
			continue
		}
		paths = append(paths, filepath.Join(m.dir, e.Name()))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		m.bot.Replace(bot.NewRegistry())
		m.mu.Lock()
		old := m.runtimes
		m.runtimes = make(map[string]*Runtime)
		m.mu.Unlock()
		for _, rt := range old {
			rt.Stop()
		}
		return nil
	}
	return m.activate(paths)
}

func (m *Manager) activate(paths []string) error {
	reg := bot.NewRegistry()
	next := make(map[string]*Runtime, len(paths))
	m.mu.RLock()
	capabilities := copyCapabilities(m.capabilities)
	permissions := copyPermissions(m.permissions)
	commandPermissions := copyCommandPermissions(m.commandPermissions)
	m.mu.RUnlock()
	for _, path := range paths {
		rt := NewLimited(path, m.bot, m.maxAllocs)
		rt.SetStore(NewStore(m.stateDir, scriptNamespace(path)))
		rt.SetHTTP(NewHTTPClient(m.httpHosts, m.httpTimeout, m.httpMaxBody))
		rt.SetCapabilities(capabilities[filepath.Base(path)])
		rt.SetPermissions(permissions)
		rt.SetCommandPermissions(commandPermissions)
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		if err := rt.prepare(src, &reg); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		rt.src = append([]byte(nil), src...)
		next[path] = rt
	}
	m.bot.Replace(reg)
	m.mu.Lock()
	old := m.runtimes
	m.runtimes = next
	m.mu.Unlock()
	for _, rt := range old {
		rt.Stop()
	}
	return nil
}

func (m *Manager) Modules() []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]map[string]any, 0, len(m.runtimes)+len(m.disabled))
	for path := range m.runtimes {
		name := filepath.Base(path)
		caps := []string{}
		c := m.capabilities[name]
		if c.HTTP {
			caps = append(caps, "http")
		}
		out = append(out, map[string]any{"id": name, "runtime": "tengo", "state": "active", "capabilities": caps})
	}
	for name, off := range m.disabled {
		if off {
			out = append(out, map[string]any{"id": name, "runtime": "tengo", "state": "disabled", "capabilities": []string{}})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	return out
}
func (m *Manager) Scripts() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.runtimes))
	for path := range m.runtimes {
		out = append(out, filepath.Base(path))
	}
	sort.Strings(out)
	return out
}
func (m *Manager) Disabled() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.disabled))
	for name, off := range m.disabled {
		if off {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
func (m *Manager) isDisabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disabled[name]
}
func copyCapabilities(in map[string]Capabilities) map[string]Capabilities {
	out := make(map[string]Capabilities, len(in))
	for name, capabilities := range in {
		out[name] = capabilities
	}
	return out
}
func copyPermissions(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for account, perms := range in {
		out[account] = append([]string(nil), perms...)
	}
	return out
}
func copyCommandPermissions(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for command, permission := range in {
		out[command] = permission
	}
	return out
}
func cleanName(name string) (string, error) {
	if filepath.Base(name) != name || !strings.HasSuffix(name, ".tengo") || name == "." {
		return "", fmt.Errorf("invalid script name %q", name)
	}
	return name, nil
}
