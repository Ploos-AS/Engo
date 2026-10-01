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

func TestStateModeOperatorAndVoice(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "", false, nil, "")
	s.Observe("JOIN", "Bob", "#x", "", false, nil, "")
	ops := s.Observe("MODE", "ChanServ", "#x", "", false, []string{"#x", "+ov", "Alice", "Bob"}, "")
	if !hasOp(ops, "assert", "channel_operator", "#x", "alice") || !hasOp(ops, "assert", "voiced", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
	ops = s.Observe("MODE", "ChanServ", "#x", "", false, []string{"#x", "-ov", "Alice", "Bob"}, "")
	if !hasOp(ops, "retract", "channel_operator", "#x", "alice") || !hasOp(ops, "retract", "voiced", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestStateNickMovesRoles(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "", false, nil, "")
	s.Observe("MODE", "Op", "#x", "", false, []string{"#x", "+ov", "Alice", "Alice"}, "")
	ops := s.Observe("NICK", "Alice", "", "", false, nil, "Bob")
	if !hasOp(ops, "retract", "channel_operator", "#x", "alice") || !hasOp(ops, "assert", "channel_operator", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
	if !hasOp(ops, "retract", "voiced", "#x", "alice") || !hasOp(ops, "assert", "voiced", "#x", "bob") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestStateKickRemovesRoles(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "", false, nil, "")
	s.Observe("MODE", "Op", "#x", "", false, []string{"#x", "+o", "Alice"}, "")
	ops := s.Observe("KICK", "Op", "#x", "", false, []string{"#x", "Alice"}, "")
	if !hasOp(ops, "retract", "channel_operator", "#x", "alice") {
		t.Fatalf("ops=%+v", ops)
	}
}

func TestNamesBootstrapMembershipAndRoles(t *testing.T) {
	s := NewIRCState()
	if ops := s.Observe("353", "server", "", "", false, []string{"engo", "=", "#x"}, "@Alice +Bob Carol"); len(ops) != 0 {
		t.Fatalf("353 mutated before 366: %+v", ops)
	}
	ops := s.Observe("366", "server", "", "", false, []string{"engo", "#x"}, "End")
	if !hasOp(ops, "assert", "channel_member", "#x", "alice") || !hasOp(ops, "assert", "channel_member", "#x", "bob") || !hasOp(ops, "assert", "channel_member", "#x", "carol") {
		t.Fatalf("membership ops=%+v", ops)
	}
	if !hasOp(ops, "assert", "channel_operator", "#x", "alice") || !hasOp(ops, "assert", "voiced", "#x", "bob") {
		t.Fatalf("role ops=%+v", ops)
	}
}
func TestNamesBootstrapReconcilesStaleState(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Old", "#x", "", false, nil, "")
	s.Observe("MODE", "Op", "#x", "", false, []string{"#x", "+o", "Old"}, "")
	s.Observe("353", "server", "", "", false, []string{"engo", "=", "#x"}, "New")
	ops := s.Observe("366", "server", "", "", false, []string{"engo", "#x"}, "End")
	if !hasOp(ops, "retract", "channel_member", "#x", "old") || !hasOp(ops, "retract", "channel_operator", "#x", "old") || !hasOp(ops, "assert", "channel_member", "#x", "new") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestNamesAccumulatesMultiple353Lines(t *testing.T) {
	s := NewIRCState()
	s.Observe("353", "server", "", "", false, []string{"engo", "=", "#x"}, "Alice Bob")
	s.Observe("353", "server", "", "", false, []string{"engo", "=", "#x"}, "Carol")
	ops := s.Observe("366", "server", "", "", false, []string{"engo", "#x"}, "End")
	for _, n := range []string{"alice", "bob", "carol"} {
		if !hasOp(ops, "assert", "channel_member", "#x", n) {
			t.Fatalf("missing %s in %+v", n, ops)
		}
	}
}

func TestStandardWHOEnrichesMembershipButNotAccount(t *testing.T) {
	s := NewIRCState()
	ops := s.Observe("352", "server", "", "", false, []string{"engo", "#x", "user", "host", "server", "Alice", "H"}, "0 Alice")
	if !hasOp(ops, "assert", "channel_member", "#x", "alice") || !hasOp(ops, "assert", "online", "alice") {
		t.Fatalf("ops=%+v", ops)
	}
	for _, op := range ops {
		if op.Fact.Predicate == "authenticated" {
			t.Fatalf("standard WHO must not assert account: %+v", ops)
		}
	}
}

func TestWHOXEnrichesVerifiedAccountFromExactContract(t *testing.T) {
	s := NewIRCState()
	ops := s.Observe("354", "server", "", "", false, []string{"engo", "152", "#x", "Alice", "AliceAcct"}, "")
	if !hasOp(ops, "assert", "channel_member", "#x", "alice") || !hasOp(ops, "assert", "authenticated", "alice", "aliceacct") {
		t.Fatalf("ops=%+v", ops)
	}
}
func TestWHOXRejectsForeignToken(t *testing.T) {
	s := NewIRCState()
	ops := s.Observe("354", "server", "", "", false, []string{"engo", "999", "#x", "Alice", "AliceAcct"}, "")
	if len(ops) != 0 {
		t.Fatalf("foreign WHOX mutated state: %+v", ops)
	}
}
func TestWHOXDoesNotAuthenticateZeroAccount(t *testing.T) {
	s := NewIRCState()
	ops := s.Observe("354", "server", "", "", false, []string{"engo", "152", "#x", "Alice", "0"}, "")
	for _, op := range ops {
		if op.Fact.Predicate == "authenticated" {
			t.Fatalf("zero account authenticated: %+v", ops)
		}
	}
}

func TestResetRetractsOwnedStateAndAllowsFreshBootstrap(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Old", "#x", "oldacct", true, nil, "")
	s.Observe("MODE", "Op", "#x", "", false, []string{"#x", "+ov", "Old", "Old"}, "")
	ops := s.Reset()
	for _, want := range []struct {
		p string
		a []string
	}{{"channel_member", []string{"#x", "old"}}, {"channel_operator", []string{"#x", "old"}}, {"voiced", []string{"#x", "old"}}, {"authenticated", []string{"old", "oldacct"}}, {"online", []string{"old"}}} {
		if !hasOp(ops, "retract", want.p, want.a...) {
			t.Fatalf("missing reset %s in %+v", want.p, ops)
		}
	}
	s.Observe("353", "server", "", "", false, []string{"engo", "=", "#x"}, "@New")
	fresh := s.Observe("366", "server", "", "", false, []string{"engo", "#x"}, "End")
	if !hasOp(fresh, "assert", "channel_member", "#x", "new") || !hasOp(fresh, "assert", "channel_operator", "#x", "new") {
		t.Fatalf("fresh=%+v", fresh)
	}
	for _, op := range fresh {
		for _, arg := range op.Fact.Args {
			if arg == "old" {
				t.Fatalf("stale state survived reset: %+v", fresh)
			}
		}
	}
}
func TestResetIsIdempotent(t *testing.T) {
	s := NewIRCState()
	s.Observe("JOIN", "Alice", "#x", "", false, nil, "")
	if len(s.Reset()) == 0 {
		t.Fatal("first reset empty")
	}
	if ops := s.Reset(); len(ops) != 0 {
		t.Fatalf("second reset=%+v", ops)
	}
}
