package utils

import (
	"runtime"
	"strings"
	"testing"
)

func TestVenvBinDir(t *testing.T) {
	got := VenvBinDir("/tmp/v")
	want := "/tmp/v/bin"
	if runtime.GOOS == "windows" {
		want = "\\tmp\\v\\Scripts"
	}
	if got != want && !strings.HasSuffix(got, "bin") && !strings.HasSuffix(got, "Scripts") {
		t.Fatalf("VenvBinDir=%q", got)
	}
}

func TestVenvExe(t *testing.T) {
	got := VenvExe("/tmp/v", "pip")
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(got, "pip.exe") {
			t.Fatalf("expected pip.exe, got %q", got)
		}
	} else {
		if !strings.HasSuffix(got, "/bin/pip") {
			t.Fatalf("expected /bin/pip suffix, got %q", got)
		}
	}
}

func TestDefaultPythonCmd(t *testing.T) {
	if got := DefaultPythonCmd("3.12"); got != "python3.12" {
		t.Fatalf("DefaultPythonCmd(3.12)=%q", got)
	}
	got := DefaultPythonCmd("")
	if runtime.GOOS == "windows" {
		if got != "python" {
			t.Fatalf("expected python, got %q", got)
		}
	} else if got != "python3" {
		t.Fatalf("expected python3, got %q", got)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{500, "500 B"},
		{2048, "2.00 KB"},
		{5 * 1024 * 1024, "5.00 MB"},
		{2 * 1024 * 1024 * 1024, "2.00 GB"},
	}
	for _, c := range cases {
		if got := FormatSize(c.in); got != c.want {
			t.Errorf("FormatSize(%d)=%q want %q", c.in, got, c.want)
		}
	}
}
