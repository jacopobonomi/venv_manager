package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const registryVersion = 1

// RegistryEntry contains persistent metadata for one managed environment.
type RegistryEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	ProjectPath string    `json:"project_path,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

type registryFile struct {
	Version      int                      `json:"version"`
	Environments map[string]RegistryEntry `json:"environments"`
}

func (m *Manager) registryPath() string {
	return filepath.Join(m.baseDir, ".venv-manager", "registry.json")
}

func newRegistry() *registryFile {
	return &registryFile{Version: registryVersion, Environments: map[string]RegistryEntry{}}
}

func (m *Manager) loadRegistry() (*registryFile, error) {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	return m.loadRegistryUnlocked()
}

func (m *Manager) loadRegistryUnlocked() (*registryFile, error) {
	data, err := os.ReadFile(m.registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return newRegistry(), nil
		}
		return nil, err
	}
	registry := newRegistry()
	if err := json.Unmarshal(data, registry); err != nil {
		return nil, fmt.Errorf("invalid registry: %w", err)
	}
	if registry.Version != registryVersion {
		return nil, fmt.Errorf("unsupported registry version %d", registry.Version)
	}
	if registry.Environments == nil {
		registry.Environments = map[string]RegistryEntry{}
	}
	return registry, nil
}

func (m *Manager) saveRegistryUnlocked(registry *registryFile) error {
	dir := filepath.Dir(m.registryPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "registry-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, m.registryPath()); err == nil {
		return nil
	} else if runtime.GOOS != "windows" {
		return err
	}
	if err := os.Remove(m.registryPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpPath, m.registryPath())
}

func (m *Manager) registerVenv(name string) error {
	now := time.Now().UTC()
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return err
	}
	entry, ok := registry.Environments[name]
	if !ok {
		entry = RegistryEntry{Name: name, Path: m.VenvPath(name), CreatedAt: now}
	}
	entry.LastUsedAt = now
	registry.Environments[name] = entry
	return m.saveRegistryUnlocked(registry)
}

func (m *Manager) touchRegistry(name string, usedAt time.Time) error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return err
	}
	entry, ok := registry.Environments[name]
	if !ok {
		entry = RegistryEntry{Name: name, Path: m.VenvPath(name), CreatedAt: usedAt}
	} else if usedAt.Sub(entry.LastUsedAt) < time.Minute {
		return nil
	}
	entry.LastUsedAt = usedAt
	registry.Environments[name] = entry
	return m.saveRegistryUnlocked(registry)
}

func (m *Manager) unregisterVenv(name string) error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return err
	}
	delete(registry.Environments, name)
	return m.saveRegistryUnlocked(registry)
}

func (m *Manager) renameRegistry(oldName, newName string) error {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return err
	}
	entry, ok := registry.Environments[oldName]
	if !ok {
		now := time.Now().UTC()
		entry = RegistryEntry{CreatedAt: now, LastUsedAt: now}
	}
	delete(registry.Environments, oldName)
	entry.Name = newName
	entry.Path = m.VenvPath(newName)
	registry.Environments[newName] = entry
	return m.saveRegistryUnlocked(registry)
}

// RegistryEntries returns metadata for all live managed environments.
func (m *Manager) RegistryEntries() ([]RegistryEntry, error) {
	venvs, err := m.List()
	if err != nil {
		return nil, err
	}
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	live := make(map[string]bool, len(venvs))
	changed := false
	for _, name := range venvs {
		live[name] = true
		if _, ok := registry.Environments[name]; !ok {
			info, statErr := os.Stat(m.VenvPath(name))
			createdAt := now
			if statErr == nil {
				createdAt = info.ModTime().UTC()
			}
			registry.Environments[name] = RegistryEntry{Name: name, Path: m.VenvPath(name), CreatedAt: createdAt, LastUsedAt: createdAt}
			changed = true
		}
	}
	for name := range registry.Environments {
		if !live[name] {
			delete(registry.Environments, name)
			changed = true
		}
	}
	if changed {
		if err := m.saveRegistryUnlocked(registry); err != nil {
			return nil, err
		}
	}
	entries := make([]RegistryEntry, 0, len(registry.Environments))
	for _, entry := range registry.Environments {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// RegistryEntry returns metadata for a single environment without touching its last-used time.
func (m *Manager) RegistryEntry(name string) (*RegistryEntry, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	entries, err := m.RegistryEntries()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Name == name {
			return &entries[i], nil
		}
	}
	return nil, fmt.Errorf("venv %q is not registered", name)
}

// SetRegistryMetadata associates a project and tags with an environment.
func (m *Manager) SetRegistryMetadata(name, projectPath string, tags []string) (*RegistryEntry, error) {
	if _, err := m.RegistryEntry(name); err != nil {
		return nil, err
	}
	if projectPath != "" {
		absolute, err := filepath.Abs(projectPath)
		if err != nil {
			return nil, err
		}
		projectPath = absolute
	}
	tags = normalizeTags(tags)
	m.registryMu.Lock()
	defer m.registryMu.Unlock()
	registry, err := m.loadRegistryUnlocked()
	if err != nil {
		return nil, err
	}
	entry := registry.Environments[name]
	entry.ProjectPath = projectPath
	entry.Tags = tags
	registry.Environments[name] = entry
	if err := m.saveRegistryUnlocked(registry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	sort.Strings(result)
	return result
}
