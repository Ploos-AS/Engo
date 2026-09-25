package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type Store struct {
	root string
	namespace string
	mu *sync.Mutex
}

var storeLocks sync.Map

func NewStore(root,namespace string)*Store{
	key:=filepath.Join(root,namespace);v,_:=storeLocks.LoadOrStore(key,&sync.Mutex{});return &Store{root:root,namespace:namespace,mu:v.(*sync.Mutex)}
}

func (s *Store) Get(key string)(string,bool,error){
	if err:=validKey(key);err!=nil{return "",false,err}
	s.mu.Lock();defer s.mu.Unlock();m,err:=s.read();if err!=nil{return "",false,err};v,ok:=m[key];return v,ok,nil
}
func (s *Store) Set(key,value string)error{
	if err:=validKey(key);err!=nil{return err};if len(value)>65536{return fmt.Errorf("KV value too large")};s.mu.Lock();defer s.mu.Unlock()
	unlock,err:=s.lockFile();if err!=nil{return err};defer unlock()
	m,err:=s.read();if err!=nil{return err};m[key]=value;return s.write(m)
}
func (s *Store) Delete(key string)error{
	if err:=validKey(key);err!=nil{return err};s.mu.Lock();defer s.mu.Unlock()
	unlock,err:=s.lockFile();if err!=nil{return err};defer unlock()
	m,err:=s.read();if err!=nil{return err};delete(m,key);return s.write(m)
}
func (s *Store) read()(map[string]string,error){
	m:=map[string]string{};if s.root==""{return m,nil};b,err:=os.ReadFile(s.path());if os.IsNotExist(err){return m,nil};if err!=nil{return nil,err};if err:=json.Unmarshal(b,&m);err!=nil{return nil,fmt.Errorf("decode store: %w",err)};return m,nil
}
func (s *Store) write(m map[string]string)error{
	if s.root==""{return nil};if err:=os.MkdirAll(s.root,0700);err!=nil{return err};b,err:=json.Marshal(m);if err!=nil{return err}
	tmp,err:=os.CreateTemp(s.root,s.namespace+".*.tmp");if err!=nil{return err}
	tmpName:=tmp.Name()
	defer os.Remove(tmpName)
	if err:=tmp.Chmod(0600);err!=nil{tmp.Close();return err}
	if _,err:=tmp.Write(b);err!=nil{tmp.Close();return err}
	if err:=tmp.Close();err!=nil{return err}
	return os.Rename(tmpName,s.path())
}
func (s *Store) path()string{return filepath.Join(s.root,s.namespace+".json")}
func validKey(key string)error{
	if key==""||len(key)>128||strings.ContainsAny(key,"\r\n"){return fmt.Errorf("invalid KV key")};return nil
}

func (s *Store) lockFile()(func(),error){
	if s.root==""{return func(){},nil}
	if err:=os.MkdirAll(s.root,0700);err!=nil{return nil,err}
	f,err:=os.OpenFile(filepath.Join(s.root,s.namespace+".lock"),os.O_CREATE|os.O_RDWR,0600)
	if err!=nil{return nil,err}
	if err:=syscall.Flock(int(f.Fd()),syscall.LOCK_EX);err!=nil{f.Close();return nil,err}
	return func(){_ = syscall.Flock(int(f.Fd()),syscall.LOCK_UN);_ = f.Close()},nil
}
