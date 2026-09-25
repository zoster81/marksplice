package parser

import "testing"

func TestBareWWWContinuationDomainAndPathOwnership(t *testing.T) {
	tests := []struct {
		name       string
		suffix     string
		wantOwner  bool
		wantEnd    int
		wantActive bool
	}{
		{name: "valid domain", suffix: "example.com", wantOwner: true, wantEnd: len("www.example.com"), wantActive: true},
		{name: "trim punctuation", suffix: "example.com*", wantOwner: true, wantEnd: len("www.example.com"), wantActive: true},
		{name: "trim close", suffix: "example.com)", wantOwner: true, wantEnd: len("www.example.com"), wantActive: true},
		{name: "invalid trailing underscore label", suffix: "example.com_x", wantOwner: false, wantActive: true},
		{name: "recover underscore label", suffix: "example.com_x.a.b", wantOwner: true, wantEnd: len("www.example.com_x.a.b"), wantActive: true},
		{name: "leading empty label", suffix: ".x.x", wantOwner: false, wantActive: false},
		{name: "hard terminator", suffix: "example.com x", wantOwner: true, wantEnd: len("www.example.com"), wantActive: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := StartBareWWWContinuation()
			active := true
			for index := 0; index < len(test.suffix); index++ {
				active = state.AppendByte(test.suffix[index])
			}
			end, ok := state.Owner()
			if ok != test.wantOwner || end != test.wantEnd {
				t.Fatalf("Owner() = (%d, %t), want (%d, %t)", end, ok, test.wantEnd, test.wantOwner)
			}
			if active != test.wantActive {
				t.Fatalf("active = %t, want %t", active, test.wantActive)
			}
		})
	}
}

func TestBareWWWContinuationReclaimsTrimmedSuffix(t *testing.T) {
	state := StartBareWWWContinuation()
	for _, b := range []byte("example.com*") {
		state.AppendByte(b)
	}
	if end, ok := state.Owner(); !ok || end != len("www.example.com") {
		t.Fatalf("base Owner() = (%d, %t)", end, ok)
	}
	if !state.AppendByte('x') {
		t.Fatal("expected continuation after reclaimed punctuation")
	}
	if end, ok := state.Owner(); !ok || end != len("www.example.com*x") {
		t.Fatalf("reclaimed Owner() = (%d, %t)", end, ok)
	}
}

func TestBareWWWContinuationFromSemanticValueAllowsReclaimedSuffix(t *testing.T) {
	state, requiredOwnerEnd, ok := BareWWWContinuationFromSemanticValue("www.example.com*")
	if !ok {
		t.Fatal("semantic value rejected")
	}
	if requiredOwnerEnd != len("www.example.com*") {
		t.Fatalf("required owner end = %d", requiredOwnerEnd)
	}
	if end, owner := state.Owner(); !owner || end != len("www.example.com") {
		t.Fatalf("materialized owner = (%d, %t)", end, owner)
	}
	if !state.AppendByte(')') {
		t.Fatal("expected latent owner continuation")
	}
	if end, owner := state.Owner(); !owner || end != requiredOwnerEnd {
		t.Fatalf("reclaimed owner = (%d, %t), want %d", end, owner, requiredOwnerEnd)
	}
}

func TestBareWWWContinuationFromAcceptedValue(t *testing.T) {
	values := []string{
		"www.example.com",
		"www.example.com/path",
		"www.example.com/a(b)c",
		"www.example.com/a&foo;x",
	}
	for _, value := range values {
		state, ok := BareWWWContinuationFromAcceptedValue(value)
		if !ok {
			t.Fatalf("BareWWWContinuationFromAcceptedValue(%q) rejected", value)
		}
		if end, owner := state.Owner(); !owner || end != len(value) {
			t.Fatalf("value %q Owner() = (%d, %t)", value, end, owner)
		}
	}
	for _, value := range []string{"", "www.", "https://example.com", "www..x.x"} {
		if _, ok := BareWWWContinuationFromAcceptedValue(value); ok {
			t.Fatalf("BareWWWContinuationFromAcceptedValue(%q) accepted", value)
		}
	}
}
