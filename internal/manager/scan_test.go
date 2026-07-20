package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractImports(t *testing.T) {
	dir := t.TempDir()
	src := `# comment
import os
import sys, json
from pathlib import Path
from collections.abc import Iterable
import numpy as np
from pandas.io import excel
from . import sibling  # relative, ignored
from .util import x    # relative, ignored
import cv2

def f():
    import requests   # nested still counted
"""
import fake_from_docstring
"""
`
	p := filepath.Join(dir, "sample.py")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := extractImports(p)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, m := range mods {
		got[m] = true
	}
	for _, want := range []string{"os", "sys", "json", "pathlib", "collections", "numpy", "pandas", "cv2", "requests"} {
		if !got[want] {
			t.Errorf("expected %q in imports, got %v", want, mods)
		}
	}
	if got["fake_from_docstring"] {
		t.Errorf("docstring content leaked into imports: %v", mods)
	}
	if got["sibling"] || got["util"] {
		t.Errorf("relative imports should be skipped: %v", mods)
	}
}

func TestPackageForImport(t *testing.T) {
	cases := map[string]string{
		"cv2":     "opencv-python",
		"sklearn": "scikit-learn",
		"PIL":     "Pillow",
		"requests": "requests",
	}
	for in, want := range cases {
		if got := PackageForImport(in); got != want {
			t.Errorf("PackageForImport(%q)=%q want %q", in, got, want)
		}
	}
}

func TestScanFiltersStdlibAndAliases(t *testing.T) {
	m, _ := newTestMgr(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.py"), []byte("import os\nimport requests\nimport cv2\n"), 0o644)
	rep, err := m.Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Files) != 1 {
		t.Fatalf("expected 1 file, got %v", rep.Files)
	}
	got := strings.Join(rep.ThirdParty, ",")
	if !strings.Contains(got, "requests") || !strings.Contains(got, "cv2") {
		t.Fatalf("third_party missing entries: %v", rep.ThirdParty)
	}
	if strings.Contains(got, "os") {
		t.Fatalf("stdlib leaked into third_party: %v", rep.ThirdParty)
	}
	suggested := strings.Join(rep.SuggestedPackages, ",")
	if !strings.Contains(suggested, "opencv-python") {
		t.Fatalf("expected opencv-python in suggested, got %v", rep.SuggestedPackages)
	}
}

func TestScanSkipsVendoredDirs(t *testing.T) {
	m, _ := newTestMgr(t)
	dir := t.TempDir()
	for _, sub := range []string{".venv", "venv", ".git", "__pycache__", "node_modules"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
		os.WriteFile(filepath.Join(dir, sub, "x.py"), []byte("import evil\n"), 0o644)
	}
	os.WriteFile(filepath.Join(dir, "good.py"), []byte("import good\n"), 0o644)
	rep, err := m.Scan(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(rep.Imports, ","), "evil") {
		t.Fatalf("scan descended into vendored dir: %v", rep.Imports)
	}
	if len(rep.Imports) != 1 || rep.Imports[0] != "good" {
		t.Fatalf("expected [good], got %v", rep.Imports)
	}
}
