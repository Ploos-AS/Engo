package botai
import("errors";"testing")
func TestStats(t *testing.T){s:=NewStats(true);s.SetCompatible(true);s.Request();s.Complete(errors.New("x"));s.Limited();m:=s.Snapshot();if m["enabled"]!=true||m["compatible"]!=true||m["requests"]!=uint64(1)||m["errors"]!=uint64(1)||m["limited"]!=uint64(1)||m["in_flight"]!=int64(0){t.Fatalf("%#v",m)}}
