package script

import (
	"fmt"\n\t"strings"\n\t"sync"
	"testing"
)

func TestStorePersists(t *testing.T) {
	root := t.TempDir()
	if err := NewStore(root, "one").Set("key", "value"); err != nil { t.Fatal(err) }
	got, ok, err := NewStore(root, "one").Get("key")
	if err != nil || !ok || got != "value" { t.Fatalf("unexpected persisted value") }
}

func TestStoreNamespaces(t *testing.T) {
	root := t.TempDir()
	if err := NewStore(root, "one").Set("key", "value"); err != nil { t.Fatal(err) }
	_, ok, err := NewStore(root, "two").Get("key")
	if err != nil || ok { t.Fatalf("namespace isolation failed") }
}

func TestStoreValueLimit(t *testing.T) {
	if err := NewStore(t.TempDir(), "one").Set("key", strings.Repeat("x", 65537)); err == nil { t.Fatal("expected size error") }
}

func TestStoreConcurrentInstances(t *testing.T){
	root:=t.TempDir()
	const n=16
	var wg sync.WaitGroup
	for i:=0;i<n;i++{
		wg.Add(1)
		go func(i int){
			defer wg.Done()
			s:=NewStore(root,"shared")
			if err:=s.Set(fmt.Sprintf("key-%d",i),fmt.Sprintf("value-%d",i));err!=nil{t.Errorf("set: %v",err)}
		}(i)
	}
	wg.Wait()
	s:=NewStore(root,"shared")
	for i:=0;i<n;i++{
		got,ok,err:=s.Get(fmt.Sprintf("key-%d",i))
		if err!=nil||!ok||got!=fmt.Sprintf("value-%d",i){t.Fatalf("missing key %d",i)}
	}
}
