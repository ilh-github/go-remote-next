package service

import (
	"testing"
)

// The default has to be the behaviour every existing install already has, or
// upgrading the server would silently start locking people out of machines
// they are already sharing.
func TestControlModeDefaultsToShared(t *testing.T) {
	if got := NormalizeControlMode(""); got != ControlModeShared {
		t.Fatalf("NormalizeControlMode(\"\") = %q, want %q", got, ControlModeShared)
	}
	for _, mode := range []string{ControlModeShared, ControlModeExclusive} {
		if got := NormalizeControlMode(mode); got != mode {
			t.Fatalf("NormalizeControlMode(%q) = %q, want it preserved", mode, got)
		}
	}
}

// An unknown mode is refused rather than silently coerced: a typo in the owner's
// settings must not quietly turn a shared machine into an exclusive one.
func TestValidateControlModeRejectsTypos(t *testing.T) {
	for _, bad := range []string{"exclusive ", "Shared", "single", "none"} {
		if err := ValidateControlMode(bad); err == nil {
			t.Fatalf("ValidateControlMode(%q) was accepted", bad)
		}
	}
}

func TestControlGrantIsExclusiveHolder(t *testing.T) {
	grant := ControlGrant{Mode: ControlModeExclusive, Holder: "cli-a"}
	if !grant.Holds("cli-a") {
		t.Fatal("the recorded holder must hold the grant")
	}
	if grant.Holds("cli-b") {
		t.Fatal("another controller was reported as the holder")
	}
}

// A shared grant is held by everybody, including a caller nobody expected: the
// point of shared mode is that nobody is turned away.
func TestSharedControlGrantIsHeldByEveryone(t *testing.T) {
	grant := ControlGrant{Mode: ControlModeShared}
	for _, who := range []string{"", "cli-a", "cli-b"} {
		if !grant.Holds(who) {
			t.Fatalf("shared mode refused %q", who)
		}
	}
}

// Exclusive mode hands the grant to whoever asks first and tells the previous
// holder it lost it. Losing the previous holder silently is the whole bug this
// prevents: without a revoke notice the old controller keeps typing into a
// machine it no longer owns.
func TestExclusiveControlGrantGoesToTheFirstAsker(t *testing.T) {
	g := NewControlGrants()
	grant := g.Request("111111111", "cli-a", ControlModeExclusive)
	if grant.Holder != "cli-a" {
		t.Fatalf("holder = %q, want cli-a", grant.Holder)
	}
	if !grant.Holds("cli-a") {
		t.Fatal("the first asker must hold the grant it was given")
	}

	second := g.Request("111111111", "cli-b", ControlModeExclusive)
	if second.Holder != "cli-b" {
		t.Fatalf("holder = %q, want the later asker cli-b", second.Holder)
	}
	if second.Holds("cli-a") {
		t.Fatal("the previous holder still holds the grant")
	}
	if _, ok := g.TakeRevoked("111111111"); !ok {
		t.Fatal("the previous holder must be told it lost the grant")
	}
}

// A second controller on a shared device must not disturb the holder, and an
// exclusive asker that already holds it keeps it without notifying anybody.
func TestSharedAndReentrantControlRequests(t *testing.T) {
	g := NewControlGrants()
	g.Request("111111111", "cli-a", ControlModeShared)
	g.Request("111111111", "cli-b", ControlModeShared)
	if _, ok := g.TakeRevoked("111111111"); ok {
		t.Fatal("shared mode produced a revoke notice")
	}

	again := g.Request("111111111", "cli-a", ControlModeExclusive)
	if again.Holder != "cli-a" {
		t.Fatalf("holder = %q, want cli-a", again.Holder)
	}
	if _, ok := g.TakeRevoked("111111111"); ok {
		t.Fatal("re-requesting a grant one already holds must not notify a revoke")
	}
}

// Releasing is idempotent and must not hand the machine to a stranger: the
// holder going away leaves the device free for the next asker, but a release
// from somebody who never held it is a no-op.
func TestControlGrantReleaseOnlyClearsTheHolder(t *testing.T) {
	g := NewControlGrants()
	g.Request("111111111", "cli-a", ControlModeExclusive)
	g.Release("111111111", "cli-b")
	if grant := g.Get("111111111"); grant.Holder != "cli-a" {
		t.Fatalf("a stranger's release moved the grant: holder = %q", grant.Holder)
	}
	g.Release("111111111", "cli-a")
	g.Release("111111111", "cli-a")
	grant := g.Request("111111111", "cli-c", ControlModeExclusive)
	if grant.Holder != "cli-c" {
		t.Fatalf("holder = %q, want the next asker cli-c", grant.Holder)
	}
}

// One device's grants are its own: a lookup or release for another device must
// not see or clear them.
func TestControlGrantsArePerDevice(t *testing.T) {
	g := NewControlGrants()
	g.Request("111111111", "cli-a", ControlModeExclusive)
	if got := g.Get("222222222"); got.Holder != "" {
		t.Fatalf("another device reported a holder: %q", got.Holder)
	}
	g.Release("222222222", "cli-a")
	if got := g.Get("111111111"); got.Holder != "cli-a" {
		t.Fatalf("a release for another device cleared the grant: holder = %q", got.Holder)
	}
}

// A device nobody has asked about has no mode, and "no mode" must not read as
// exclusive-with-no-holder - that would freeze a machine nobody configured.
func TestControlGrantForAnUnknownDeviceHoldsForEverybody(t *testing.T) {
	g := NewControlGrants()
	grant := g.Get("111111111")
	if grant.Mode == ControlModeExclusive {
		t.Fatal("an unconfigured device was reported as exclusive")
	}
	for _, who := range []string{"", "cli-a", "cli-b"} {
		if !grant.Holds(who) {
			t.Fatalf("an unconfigured device refused %q", who)
		}
	}
}

// Releasing an exclusive machine frees it without turning it into a shared one:
// the next controller may take it, but only by asking, which is what keeps a
// released machine from silently going back to everybody-at-once.
func TestReleasedExclusiveGrantStaysExclusiveWithNoHolder(t *testing.T) {
	g := NewControlGrants()
	g.Request("111111111", "cli-a", ControlModeExclusive)
	g.Release("111111111", "cli-a")
	grant := g.Get("111111111")
	if grant.Mode != ControlModeExclusive {
		t.Fatalf("mode = %q after the holder released, want %q", grant.Mode, ControlModeExclusive)
	}
	if grant.Holder != "" {
		t.Fatalf("holder = %q after the holder released, want empty", grant.Holder)
	}
	if grant.Holds("cli-b") {
		t.Fatal("a released exclusive machine handed control to a controller that never asked")
	}
}

// Removing a device drops its pending notices too, or a controller that never
// comes back would be notified about a grant it can no longer take.
func TestControlGrantForgetDeviceClearsNotices(t *testing.T) {
	g := NewControlGrants()
	g.Request("111111111", "cli-a", ControlModeExclusive)
	g.Request("111111111", "cli-b", ControlModeExclusive)
	g.ForgetDevice("111111111")
	if _, ok := g.TakeRevoked("111111111"); ok {
		t.Fatal("a forgotten device still had a pending revoke notice")
	}
	if got := g.Get("111111111"); got.Holder != "" {
		t.Fatalf("a forgotten device still reported holder %q", got.Holder)
	}
}
