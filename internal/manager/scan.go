package manager

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// stdlibModules is a set of well-known Python stdlib top-level modules used
// to filter scan results. Not exhaustive but covers ~all common cases.
var stdlibModules = map[string]bool{
	"__future__": true, "_thread": true, "abc": true, "argparse": true, "array": true,
	"ast": true, "asyncio": true, "atexit": true, "base64": true, "binascii": true,
	"bisect": true, "builtins": true, "bz2": true, "calendar": true, "cmath": true,
	"cmd": true, "codecs": true, "collections": true, "colorsys": true, "concurrent": true,
	"configparser": true, "contextlib": true, "copy": true, "csv": true, "ctypes": true,
	"dataclasses": true, "datetime": true, "decimal": true, "difflib": true, "dis": true,
	"email": true, "encodings": true, "enum": true, "errno": true, "faulthandler": true,
	"filecmp": true, "fileinput": true, "fnmatch": true, "fractions": true, "ftplib": true,
	"functools": true, "gc": true, "getopt": true, "getpass": true, "gettext": true,
	"glob": true, "gzip": true, "hashlib": true, "heapq": true, "hmac": true, "html": true,
	"http": true, "imaplib": true, "importlib": true, "inspect": true, "io": true,
	"ipaddress": true, "itertools": true, "json": true, "keyword": true, "linecache": true,
	"locale": true, "logging": true, "lzma": true, "math": true, "mimetypes": true,
	"multiprocessing": true, "netrc": true, "numbers": true, "operator": true, "os": true,
	"pathlib": true, "pickle": true, "platform": true, "plistlib": true, "posixpath": true,
	"pprint": true, "profile": true, "pstats": true, "queue": true, "random": true,
	"re": true, "resource": true, "sched": true, "secrets": true, "select": true,
	"selectors": true, "shelve": true, "shlex": true, "shutil": true, "signal": true,
	"site": true, "smtplib": true, "socket": true, "socketserver": true, "sqlite3": true,
	"ssl": true, "stat": true, "string": true, "stringprep": true, "struct": true,
	"subprocess": true, "sys": true, "sysconfig": true, "tarfile": true, "tempfile": true,
	"textwrap": true, "threading": true, "time": true, "timeit": true, "tkinter": true,
	"token": true, "tokenize": true, "trace": true, "traceback": true, "types": true,
	"typing": true, "unicodedata": true, "unittest": true, "urllib": true, "uuid": true,
	"venv": true, "warnings": true, "weakref": true, "webbrowser": true, "wsgiref": true,
	"xml": true, "xmlrpc": true, "zipfile": true, "zipimport": true, "zlib": true,
	"zoneinfo": true,
}

// importAliases maps import names that differ from their pip package name.
var importAliases = map[string]string{
	"cv2":      "opencv-python",
	"sklearn":  "scikit-learn",
	"PIL":      "Pillow",
	"skimage":  "scikit-image",
	"yaml":     "PyYAML",
	"bs4":      "beautifulsoup4",
	"dateutil": "python-dateutil",
	"dotenv":   "python-dotenv",
	"magic":    "python-magic",
	"jose":     "python-jose",
	"jwt":      "PyJWT",
	"attr":     "attrs",
	"OpenSSL":  "pyOpenSSL",
	"google":   "google-cloud-core",
}

// pep503SepRe collapses runs of '-', '_' and '.' as per PEP 503.
var pep503SepRe = regexp.MustCompile(`[-_.]+`)

// normalizePkgName applies PEP 503 normalization so that names like
// "typing_extensions" and "Typing-Extensions" compare equal.
func normalizePkgName(name string) string {
	return strings.ToLower(pep503SepRe.ReplaceAllString(name, "-"))
}

// PackageForImport returns the pip package name for a given import module.
func PackageForImport(mod string) string {
	if p, ok := importAliases[mod]; ok {
		return p
	}
	return mod
}

var (
	reImport     = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][\w\.]*(?:\s*,\s*[a-zA-Z_][\w\.]*)*)`)
	reFromImport = regexp.MustCompile(`^\s*from\s+([a-zA-Z_][\w\.]*)\s+import\s+`)
)

// ScanReport summarizes imports discovered in a path and, if a venv name is
// given, which imports are satisfied by that venv.
type ScanReport struct {
	Path              string   `json:"path"`
	Files             []string `json:"files"`
	Imports           []string `json:"imports"`            // top-level module names, deduped
	ThirdParty        []string `json:"third_party"`        // excluding stdlib
	SuggestedPackages []string `json:"suggested_packages"` // pip names (with alias resolution)
	Venv              string   `json:"venv,omitempty"`
	Missing           []string `json:"missing,omitempty"` // packages not installed in the given venv
}

// Scan walks a file or directory tree, extracts top-level Python imports, and
// (optionally) checks which are missing in the named venv.
func (m *Manager) Scan(path, venv string) (*ScanReport, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files []string
	if info.IsDir() {
		err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				name := fi.Name()
				if name == ".venv" || name == "venv" || name == ".git" || name == "__pycache__" ||
					name == "node_modules" || name == "site-packages" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".py") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		files = []string{path}
	}

	seen := map[string]bool{}
	for _, f := range files {
		mods, err := extractImports(f)
		if err != nil {
			continue
		}
		for _, m := range mods {
			seen[m] = true
		}
	}

	imports := sortedKeys(seen)
	var thirdParty, suggested []string
	for _, imp := range imports {
		if stdlibModules[imp] || strings.HasPrefix(imp, "_") {
			continue
		}
		thirdParty = append(thirdParty, imp)
		suggested = append(suggested, PackageForImport(imp))
	}

	rep := &ScanReport{
		Path:              path,
		Files:             files,
		Imports:           imports,
		ThirdParty:        thirdParty,
		SuggestedPackages: suggested,
	}
	if venv != "" {
		installed, err := m.ListPackages(venv)
		if err != nil {
			return rep, err
		}
		installedSet := map[string]bool{}
		for _, spec := range installed {
			name := strings.SplitN(spec, "==", 2)[0]
			installedSet[normalizePkgName(name)] = true
		}
		for _, pkg := range suggested {
			if !installedSet[normalizePkgName(pkg)] {
				rep.Missing = append(rep.Missing, pkg)
			}
		}
		rep.Venv = venv
	}
	return rep, nil
}

func extractImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var mods []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	inTriple := false
	tripleDelim := ""
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if inTriple {
			if strings.Contains(trim, tripleDelim) {
				inTriple = false
			}
			continue
		}
		if strings.HasPrefix(trim, `"""`) || strings.HasPrefix(trim, `'''`) {
			d := trim[:3]
			rest := trim[3:]
			if !strings.Contains(rest, d) {
				inTriple = true
				tripleDelim = d
			}
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		if match := reFromImport.FindStringSubmatch(line); match != nil {
			mod := match[1]
			if !strings.HasPrefix(mod, ".") {
				mods = append(mods, topModule(mod))
			}
			continue
		}
		if match := reImport.FindStringSubmatch(line); match != nil {
			for _, part := range strings.Split(match[1], ",") {
				part = strings.TrimSpace(part)
				if strings.Contains(part, " as ") {
					part = strings.SplitN(part, " as ", 2)[0]
				}
				part = strings.TrimSpace(part)
				if part != "" && !strings.HasPrefix(part, ".") {
					mods = append(mods, topModule(part))
				}
			}
		}
	}
	return mods, sc.Err()
}

func topModule(m string) string {
	if idx := strings.Index(m, "."); idx > 0 {
		return m[:idx]
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// summary is used by the CLI to print a human-friendly line.
func (r *ScanReport) Summary() string {
	return fmt.Sprintf("scanned %d file(s), %d imports (%d third-party)",
		len(r.Files), len(r.Imports), len(r.ThirdParty))
}
