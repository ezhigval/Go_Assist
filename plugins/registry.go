package plugins

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"modulr/events"
)

var (
	// ErrManifestIDRequired означает отсутствие plugin id.
	ErrManifestIDRequired = errors.New("plugins: manifest id required")
	// ErrManifestVersionRequired означает отсутствие версии.
	ErrManifestVersionRequired = errors.New("plugins: manifest version required")
	// ErrManifestRuntimeRequired означает отсутствие runtime.
	ErrManifestRuntimeRequired = errors.New("plugins: manifest runtime required")
	// ErrManifestEntryRequired означает отсутствие entrypoint.
	ErrManifestEntryRequired = errors.New("plugins: manifest entry required")
	// ErrManifestEntryAbsolute означает, что manifest использует абсолютный entry path.
	ErrManifestEntryAbsolute = errors.New("plugins: absolute entry is not allowed")
	// ErrManifestEntryPathMismatch означает, что resolved entry path не совпадает с manifest entry/source path.
	ErrManifestEntryPathMismatch = errors.New("plugins: entry path mismatch")
	// ErrManifestCapabilityRequired означает отсутствие capabilities.
	ErrManifestCapabilityRequired = errors.New("plugins: manifest capability required")
	// ErrManifestEntryEscapesDir означает, что относительный entry выходит за границы каталога manifest.
	ErrManifestEntryEscapesDir = errors.New("plugins: entry escapes manifest directory")
	// ErrDuplicateManifest означает повторную регистрацию manifest key.
	ErrDuplicateManifest = errors.New("plugins: duplicate manifest")
)

var (
	manifestIDPattern      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	manifestVersionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+([\-+][0-9A-Za-z.\-]+)?$`)
	capabilityNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	allowedPermissions     = map[string]struct{}{
		"env.read":         {},
		"fs.read":          {},
		"fs.write":         {},
		"network.outbound": {},
	}
)

// Runtime определяет способ исполнения плагина.
type Runtime string

const (
	// RuntimeProcess — fallback-путь v2.0: отдельный OS-process.
	RuntimeProcess Runtime = "process"
	// RuntimeWASM — sandbox runtime для будущего WASM path.
	RuntimeWASM Runtime = "wasm"
)

// Protocol определяет способ общения host ↔ plugin.
type Protocol string

const (
	ProtocolGRPC  Protocol = "grpc"
	ProtocolStdio Protocol = "stdio"
)

// Capability описывает, за какие module/action pairs отвечает plugin.
type Capability struct {
	Module  string   `json:"module"`
	Actions []string `json:"actions"`
	Scopes  []string `json:"scopes,omitempty"`
}

// Manifest — версия plugin contract для v2.0.
type Manifest struct {
	ID           string       `json:"id"`
	Version      string       `json:"version"`
	DisplayName  string       `json:"display_name,omitempty"`
	Description  string       `json:"description,omitempty"`
	Runtime      Runtime      `json:"runtime"`
	Protocol     Protocol     `json:"protocol,omitempty"`
	Entry        string       `json:"entry"`
	Permissions  []string     `json:"permissions,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Capabilities []Capability `json:"capabilities"`
}

// LoadedManifest — manifest с уже разрешённым путём до entrypoint.
type LoadedManifest struct {
	Manifest
	SourcePath string `json:"source_path"`
	EntryPath  string `json:"entry_path"`
}

// Registry хранит versioned plugin manifests и умеет резолвить capability lookup.
type Registry struct {
	mu        sync.RWMutex
	manifests map[string]LoadedManifest
}

// NewRegistry создаёт пустой plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		manifests: make(map[string]LoadedManifest),
	}
}

// ParseManifest валидирует JSON manifest без привязки к файловой системе.
func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("plugins: decode manifest: %w", err)
	}
	normalizeManifest(&manifest)
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// LoadManifest читает и валидирует один `.plugin.json`.
func LoadManifest(path string) (LoadedManifest, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return LoadedManifest{}, fmt.Errorf("plugins: abs path: %w", err)
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return LoadedManifest{}, fmt.Errorf("plugins: read manifest: %w", err)
	}
	manifest, err := ParseManifest(raw)
	if err != nil {
		return LoadedManifest{}, err
	}
	entryPath, err := resolveEntryPath(filepath.Dir(absPath), manifest.Entry)
	if err != nil {
		return LoadedManifest{}, err
	}
	return LoadedManifest{
		Manifest:   manifest,
		SourcePath: absPath,
		EntryPath:  entryPath,
	}, nil
}

// LoadDir загружает все manifests из каталога по шаблону `*.plugin.json`.
func LoadDir(dir string) ([]LoadedManifest, error) {
	pattern := filepath.Join(dir, "*.plugin.json")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("plugins: glob manifests: %w", err)
	}
	sort.Strings(paths)

	loaded := make([]LoadedManifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := LoadManifest(path)
		if err != nil {
			return nil, err
		}
		loaded = append(loaded, manifest)
	}
	return loaded, nil
}

// Register добавляет manifest в registry; ключ — `id@version`.
func (r *Registry) Register(manifest LoadedManifest) error {
	if err := validateLoadedManifest(manifest); err != nil {
		return err
	}
	key := manifest.Key()
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.manifests[key]; exists {
		return ErrDuplicateManifest
	}
	r.manifests[key] = manifest
	return nil
}

// List возвращает manifests в стабильном порядке.
func (r *Registry) List() []LoadedManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.manifests))
	for key := range r.manifests {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]LoadedManifest, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.manifests[key])
	}
	return out
}

// Resolve возвращает manifests, которые поддерживают module/action.
func (r *Registry) Resolve(module, action string) []LoadedManifest {
	module = strings.ToLower(strings.TrimSpace(module))
	action = strings.ToLower(strings.TrimSpace(action))
	if module == "" || action == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.manifests))
	for key, manifest := range r.manifests {
		if manifest.Supports(module, action) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	out := make([]LoadedManifest, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.manifests[key])
	}
	return out
}

// Key возвращает versioned идентификатор manifest.
func (m Manifest) Key() string {
	return fmt.Sprintf("%s@%s", m.ID, m.Version)
}

// Key возвращает versioned идентификатор loaded manifest.
func (m LoadedManifest) Key() string {
	return m.Manifest.Key()
}

// Supports true, если plugin умеет обрабатывать pair module/action.
func (m Manifest) Supports(module, action string) bool {
	module = strings.ToLower(strings.TrimSpace(module))
	action = strings.ToLower(strings.TrimSpace(action))
	for _, capability := range m.Capabilities {
		if capability.Module != module {
			continue
		}
		for _, candidate := range capability.Actions {
			if candidate == action {
				return true
			}
		}
	}
	return false
}

func normalizeManifest(manifest *Manifest) {
	manifest.ID = strings.ToLower(strings.TrimSpace(manifest.ID))
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.DisplayName = strings.TrimSpace(manifest.DisplayName)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Runtime = Runtime(strings.ToLower(strings.TrimSpace(string(manifest.Runtime))))
	manifest.Protocol = Protocol(strings.ToLower(strings.TrimSpace(string(manifest.Protocol))))
	manifest.Entry = strings.TrimSpace(manifest.Entry)
	manifest.Permissions = normalizeStrings(manifest.Permissions, true)
	manifest.Tags = normalizeStrings(manifest.Tags, false)
	for i := range manifest.Capabilities {
		manifest.Capabilities[i].Module = strings.ToLower(strings.TrimSpace(manifest.Capabilities[i].Module))
		manifest.Capabilities[i].Actions = normalizeStrings(manifest.Capabilities[i].Actions, true)
		manifest.Capabilities[i].Scopes = normalizeStrings(manifest.Capabilities[i].Scopes, true)
	}
}

func validateManifest(manifest Manifest) error {
	if manifest.ID == "" {
		return ErrManifestIDRequired
	}
	if !manifestIDPattern.MatchString(manifest.ID) {
		return fmt.Errorf("plugins: invalid manifest id %q", manifest.ID)
	}
	if manifest.Version == "" {
		return ErrManifestVersionRequired
	}
	if !manifestVersionPattern.MatchString(manifest.Version) {
		return fmt.Errorf("plugins: invalid manifest version %q", manifest.Version)
	}
	if manifest.Runtime == "" {
		return ErrManifestRuntimeRequired
	}
	switch manifest.Runtime {
	case RuntimeProcess:
		if manifest.Protocol != ProtocolGRPC && manifest.Protocol != ProtocolStdio {
			return fmt.Errorf("plugins: process manifest %q must declare protocol grpc|stdio", manifest.ID)
		}
	case RuntimeWASM:
		if manifest.Protocol != "" && manifest.Protocol != ProtocolStdio {
			return fmt.Errorf("plugins: wasm manifest %q must use stdio protocol when protocol is set", manifest.ID)
		}
		if !strings.HasSuffix(strings.ToLower(manifest.Entry), ".wasm") {
			return fmt.Errorf("plugins: wasm manifest %q must point to .wasm entry", manifest.ID)
		}
	default:
		return fmt.Errorf("plugins: unsupported runtime %q", manifest.Runtime)
	}
	if manifest.Entry == "" {
		return ErrManifestEntryRequired
	}
	if filepath.IsAbs(manifest.Entry) {
		return ErrManifestEntryAbsolute
	}
	if len(manifest.Capabilities) == 0 {
		return ErrManifestCapabilityRequired
	}
	for _, permission := range manifest.Permissions {
		if _, ok := allowedPermissions[permission]; !ok {
			return fmt.Errorf("plugins: unsupported permission %q", permission)
		}
	}
	for _, capability := range manifest.Capabilities {
		if !capabilityNamePattern.MatchString(capability.Module) {
			return fmt.Errorf("plugins: invalid capability module %q", capability.Module)
		}
		if len(capability.Actions) == 0 {
			return ErrManifestCapabilityRequired
		}
		for _, action := range capability.Actions {
			if !capabilityNamePattern.MatchString(action) {
				return fmt.Errorf("plugins: invalid capability action %q", action)
			}
		}
		for _, scope := range capability.Scopes {
			if !events.IsValidSegment(events.Segment(scope)) {
				return fmt.Errorf("plugins: invalid capability scope %q", scope)
			}
		}
	}
	return nil
}

func validateLoadedManifest(manifest LoadedManifest) error {
	if err := validateManifest(manifest.Manifest); err != nil {
		return err
	}
	if strings.TrimSpace(manifest.SourcePath) == "" || strings.TrimSpace(manifest.EntryPath) == "" {
		return nil
	}

	absSource, err := filepath.Abs(manifest.SourcePath)
	if err != nil {
		return fmt.Errorf("plugins: abs source path: %w", err)
	}
	expectedEntry, err := resolveEntryPath(filepath.Dir(absSource), manifest.Entry)
	if err != nil {
		return err
	}
	if filepath.Clean(manifest.EntryPath) != expectedEntry {
		return ErrManifestEntryPathMismatch
	}
	return nil
}

func resolveEntryPath(baseDir, entry string) (string, error) {
	if strings.TrimSpace(entry) == "" {
		return "", ErrManifestEntryRequired
	}
	if filepath.IsAbs(entry) {
		return "", ErrManifestEntryAbsolute
	}
	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("plugins: abs base dir: %w", err)
	}
	target, err := filepath.Abs(filepath.Join(baseDir, entry))
	if err != nil {
		return "", fmt.Errorf("plugins: abs entry path: %w", err)
	}
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", fmt.Errorf("plugins: relative entry path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrManifestEntryEscapesDir
	}
	return filepath.Clean(target), nil
}

func normalizeStrings(values []string, lower bool) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
