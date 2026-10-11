package cobrax

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

// Source is a value source that can be checked for a flag's value.
type Source interface {
	Lookup() (string, bool)
	String() string
}

// Env returns a Source that reads from an environment variable.
func Env(name string) Source {
	return envSource(name)
}

type envSource string

func (e envSource) Lookup() (string, bool) {
	v := os.Getenv(string(e))
	return v, v != ""
}

func (e envSource) String() string {
	return "$" + string(e)
}

// SecretFile returns a Source that reads a file at filepath.Join(*mount, name).
// The mount pointer is evaluated lazily at Resolve time.
func SecretFile(mount *string, name string) Source {
	return &secretFileSource{mount: mount, name: name}
}

type secretFileSource struct {
	mount *string
	name  string
}

func (s *secretFileSource) Lookup() (string, bool) {
	if s.mount == nil || *s.mount == "" {
		return "", false
	}
	data, err := os.ReadFile(filepath.Join(*s.mount, s.name))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func (s *secretFileSource) String() string {
	return "file " + s.name
}

// Flags wraps a pflag.FlagSet and tracks source chains for lazy resolution.
type Flags struct {
	*pflag.FlagSet
	sources map[string][]Source
}

// NewFlags creates a Flags wrapper around the given FlagSet.
func NewFlags(fs *pflag.FlagSet) *Flags {
	return &Flags{FlagSet: fs}
}

func (f *Flags) addSources(name string, sources []Source) {
	if len(sources) == 0 {
		return
	}
	if f.sources == nil {
		f.sources = make(map[string][]Source)
	}
	f.sources[name] = sources
}

func sourceUsage(sources []Source) string {
	if len(sources) == 0 {
		return ""
	}
	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = s.String()
	}
	return " [" + strings.Join(names, ", ") + "]"
}

// String registers a string flag with optional source chain fallbacks.
// Sources are evaluated lazily when Resolve is called.
func (f *Flags) String(p *string, name, short, def, usage string, sources ...Source) {
	f.StringVarP(p, name, short, def, usage+sourceUsage(sources))
	f.addSources(name, sources)
}

// Bool registers a bool flag with optional source chain fallbacks.
// Sources are evaluated lazily when Resolve is called.
func (f *Flags) Bool(p *bool, name, short string, def bool, usage string, sources ...Source) {
	f.BoolVarP(p, name, short, def, usage+sourceUsage(sources))
	f.addSources(name, sources)
}

// Resolve walks all flags with source chains and applies the first matching
// source value for any flag that was not explicitly set on the command line.
func (f *Flags) Resolve() error {
	for name, sources := range f.sources {
		flag := f.Lookup(name)
		if flag == nil || flag.Changed {
			continue
		}
		for _, src := range sources {
			if v, found := src.Lookup(); found {
				if err := flag.Value.Set(v); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}
