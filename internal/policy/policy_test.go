package policy

import "testing"

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy("/repo", "/home/user")
	if p.Network != NetworkOn {
		t.Fatalf("expected network on")
	}
	if p.Resources.CPUs != DefaultCPUs || p.Resources.MemoryMB != DefaultMemoryMB {
		t.Fatalf("unexpected defaults: %+v", p.Resources)
	}
	if p.Unsafe {
		t.Fatalf("expected unsafe false by default")
	}
}

func TestMountRules(t *testing.T) {
	p := DefaultPolicy("/repo", "/home/user")

	cases := []struct {
		path   string
		allow  bool
		unsafe bool
	}{
		{"/repo", true, false},
		{"/repo/subdir", true, false},
		{"/home/user", false, false},
		{"/var/run/docker.sock", false, false},
		{"/", false, false},
		{"/etc", false, false},
		{"/home/user", true, true},
		{"/etc", true, true},
	}

	for _, tc := range cases {
		p.Unsafe = tc.unsafe
		err := ValidateMount(p, tc.path)
		if tc.allow && err != nil {
			t.Fatalf("expected allow for %q unsafe=%v, got %v", tc.path, tc.unsafe, err)
		}
		if !tc.allow && err == nil {
			t.Fatalf("expected deny for %q unsafe=%v", tc.path, tc.unsafe)
		}
	}
}
