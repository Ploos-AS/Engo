package botlogic

import "testing"

func hasOp(ops []FactOperation, op, p string, args ...string) bool {
	for _, x := range ops {
		if x.Op != op || x.Fact.Predicate != p || len(x.Fact.Args) != len(args) {
			continue
		}
		ok := true
		for i := range args {
			if x.Fact.Args[i] != args[i] {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func TestStateQuitRetractsAllMemberships(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#b", "acct", true, nil, "")
	s.Observe("JOIN", "Alice", "#a", "acct", true, nil, "")
	ops := s.Observe("QUIT", "Alice", "", "", false, nil, "")
	if !hasOp(ops, "retract", "channel_member", "#a", "alice") || !hasOp(ops, "retract", "channel_member", "#b", "alice") {
		t.Fatalf("ops=%+v", ops)
	}
	if !hasOp(ops, "retract", "online", "alice") || !hasOp(ops, "retract", "authenticated", "alice", "acct") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestStateNickMovesMembershipAndAccount(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "acct", true, nil, "")
	ops := s.Observe("NICK", "Alice", "", "", false, nil, "Bob")
	if !hasOp(ops, "retract", "channel_member", "#x", "alice") || !hasOp(ops, "assert", "channel_member", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
	if !hasOp(ops, "retract", "authenticated", "alice", "acct") || !hasOp(ops, "assert", "authenticated", "bob", "acct") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestStateKickRetractsVictimMembership(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Bob", "#x", "", false, nil, "")
	ops := s.Observe("KICK", "Op", "#x", "", false, []string{"#x", "Bob"}, "reason")
	if !hasOp(ops, "retract", "channel_member", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestStateAccountLogoutRetractsKnownAccount(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "acct", true, nil, "")
	ops := s.Observe("ACCOUNT", "Alice", "", "", false, []string{"*"}, "")
	if !hasOp(ops, "retract", "authenticated", "alice", "acct") {
		t.Fatalf("ops=%+v", ops)
	}
}
