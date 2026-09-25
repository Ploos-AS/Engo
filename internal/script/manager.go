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
	maxAllocs int64\n\tstateDir string
	mu sync.RWMutex
	runtimes map[string]*Runtime
	disabled map[string]bool
}

func NewManager(dir string,b *bot.Bot)*Manager{return NewManagerLimited(dir,b,100000)}
func NewManagerLimited(dir string,b *bot.Bot,maxAllocs int64)*Manager{
	return &Manager{dir:dir,bot:b,maxAllocs:maxAllocs,runtimes:make(map[string]*Runtime),disabled:make(map[string]bool)}
}

func (m *Manager) ReloadAll() error {
	entries,err:=os.ReadDir(m.dir); if err!=nil{return fmt.Errorf("read scripts directory: %w",err)}
	var paths []string
	for _,e:=range entries{
		if e.IsDir()||!strings.HasSuffix(e.Name(),".tengo")||m.isDisabled(e.Name()){continue}
		paths=append(paths,filepath.Join(m.dir,e.Name()))
	}
	sort.Strings(paths)
	if len(paths)==0{return fmt.Errorf("no enabled .tengo scripts in %s",m.dir)}
	return m.activate(paths)
}

func (m *Manager) Enable(name string) error {
	name,err:=cleanName(name); if err!=nil{return err}
	path:=filepath.Join(m.dir,name)
	if _,err:=os.Stat(path);err!=nil{return err}
	m.mu.Lock(); delete(m.disabled,name); m.mu.Unlock()
	if err:=m.reloadCurrent();err!=nil{
		m.mu.Lock();m.disabled[name]=true;m.mu.Unlock()
		return err
	}
	return nil
}

func (m *Manager) Disable(name string) error {
	name,err:=cleanName(name);if err!=nil{return err}
	m.mu.Lock();m.disabled[name]=true;m.mu.Unlock()
	if err:=m.reloadCurrent();err!=nil{
		m.mu.Lock();delete(m.disabled,name);m.mu.Unlock()
		return err
	}
	return nil
}

func (m *Manager) Reload(name string) error {
	name,err:=cleanName(name);if err!=nil{return err}
	if m.isDisabled(name){return fmt.Errorf("%s is disabled",name)}
	return m.reloadCurrent()
}

func (m *Manager) reloadCurrent() error {
	entries,err:=os.ReadDir(m.dir);if err!=nil{return err}
	var paths []string
	for _,e:=range entries{
		if e.IsDir()||!strings.HasSuffix(e.Name(),".tengo")||m.isDisabled(e.Name()){continue}
		paths=append(paths,filepath.Join(m.dir,e.Name()))
	}
	sort.Strings(paths)
	if len(paths)==0{
		m.bot.Replace(bot.NewRegistry())
		m.mu.Lock();m.runtimes=make(map[string]*Runtime);m.mu.Unlock()
		return nil
	}
	return m.activate(paths)
}

func (m *Manager) activate(paths []string) error {
	reg:=bot.NewRegistry();next:=make(map[string]*Runtime,len(paths))
	for _,path:=range paths{
		rt:=NewLimited(path,m.bot,m.maxAllocs)\n\t\trt.SetStore(NewStore(m.stateDir,scriptNamespace(path)))
		src,err:=os.ReadFile(path);if err!=nil{return fmt.Errorf("%s: %w",filepath.Base(path),err)}
		if err:=rt.prepare(src,&reg);err!=nil{return fmt.Errorf("%s: %w",filepath.Base(path),err)}
		rt.src=append([]byte(nil),src...);next[path]=rt
	}
	m.bot.Replace(reg);m.mu.Lock();m.runtimes=next;m.mu.Unlock();return nil
}

func (m *Manager) Scripts()[]string{
	m.mu.RLock();defer m.mu.RUnlock();out:=make([]string,0,len(m.runtimes))
	for path:=range m.runtimes{out=append(out,filepath.Base(path))};sort.Strings(out);return out
}
func (m *Manager) Disabled()[]string{
	m.mu.RLock();defer m.mu.RUnlock();out:=make([]string,0,len(m.disabled))
	for name,off:=range m.disabled{if off{out=append(out,name)}};sort.Strings(out);return out
}
func (m *Manager) isDisabled(name string)bool{m.mu.RLock();defer m.mu.RUnlock();return m.disabled[name]}
func cleanName(name string)(string,error){
	if filepath.Base(name)!=name||!strings.HasSuffix(name,".tengo")||name=="."{return "",fmt.Errorf("invalid script name %q",name)}
	return name,nil
}
