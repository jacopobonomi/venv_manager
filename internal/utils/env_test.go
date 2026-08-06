package utils

import (
	"strings"
	"testing"
)

func TestSetEnv_new(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := SetEnv(env, "C", "3")
	if last := got[len(got)-1]; last != "C=3" {
		t.Fatalf("expected C=3 appended, got %v", got)
	}
}

func TestSetEnv_replace(t *testing.T) {
	env := []string{"A=1", "B=2", "C=old"}
	got := SetEnv(env, "C", "new")
	for _, e := range got {
		if strings.HasPrefix(e, "C=") && e != "C=new" {
			t.Fatalf("expected C=new, got %v", got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("expected same length, got %v", got)
	}
}

func TestSetEnv_preservesOthers(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := SetEnv(env, "A", "99")
	if got[0] != "A=99" || got[1] != "B=2" {
		t.Fatalf("unexpected env %v", got)
	}
}

func TestRemoveEnv_present(t *testing.T) {
	env := []string{"A=1", "B=2", "C=3"}
	got := RemoveEnv(env, "B")
	for _, e := range got {
		if strings.HasPrefix(e, "B=") {
			t.Fatalf("B should be removed, got %v", got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %v", got)
	}
}

func TestRemoveEnv_missing(t *testing.T) {
	env := []string{"A=1", "B=2"}
	got := RemoveEnv(env, "C")
	if len(got) != 2 {
		t.Fatalf("expected unchanged, got %v", got)
	}
}

func TestRemoveEnv_empty(t *testing.T) {
	got := RemoveEnv(nil, "A")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
