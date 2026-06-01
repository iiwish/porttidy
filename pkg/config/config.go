package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultDevSignatures are built-in patterns for identifying dev processes
var DefaultDevSignatures = []string{
	"vite", "webpack serve", "rollup", "esbuild", "next dev", "nuxt dev", "astro dev",
	"python -m http.server", "python3 -m http.server",
	"serve", "http-server",
	"python manage.py runserver", "uvicorn", "fastapi", "flask run",
}

// DefaultDenylist are processes that should never be killed
var DefaultDenylist = []string{
	"Code", "Code Helper", "Code Helper (Renderer)", "Code Helper (GPU)", "Code Helper (Plugin)",
	"Google Chrome", "Google Chrome Helper", "Google Chrome Helper (Renderer)", "Google Chrome Helper (GPU)",
	"Linear", "Linear Helper", "Linear Helper (Renderer)", "Linear Helper (GPU)",
	"Clash Verge", "clash-verge", "verge-mihomo",
	"WeChat", "微信",
	"Kimi Code", "Codex", "node_repl", "Claude", "Claude Code", "opencode", "Cursor",
	"Xcode",
	"Finder", "Dock", "SystemUIServer", "WindowServer",
	"Slack", "Discord", "Telegram",
	"Docker Desktop", "docker",
	"iTerm", "Terminal", "Warp", "Alacritty",
	"rapportd", "ControlCe",
}

// Config holds porttidy configuration
type Config struct {
	TargetDirs    []string `yaml:"target_dirs"`
	DevSignatures []string `yaml:"dev_signatures"`
	Denylist      []string `yaml:"denylist"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		TargetDirs: []string{
			filepath.Join(os.Getenv("HOME"), "daas"),
			filepath.Join(os.Getenv("HOME"), "self"),
		},
		DevSignatures: DefaultDevSignatures,
		Denylist:      DefaultDenylist,
	}
}

// ConfigDir returns the configuration directory path
func ConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "porttidy")
}

// ConfigPath returns the full path to the config file
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "porttidy.yaml")
}

// Load reads configuration from file or creates default
func Load(path string) (*Config, error) {
	if path == "" {
		path = ConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := cfg.Save(path); err != nil {
				return cfg, fmt.Errorf("creating default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Merge with defaults for empty slices
	if len(cfg.TargetDirs) == 0 {
		cfg.TargetDirs = DefaultConfig().TargetDirs
	}
	if len(cfg.DevSignatures) == 0 {
		cfg.DevSignatures = DefaultDevSignatures
	}
	if len(cfg.Denylist) == 0 {
		cfg.Denylist = DefaultDenylist
	} else {
		cfg.Denylist = mergeUnique(DefaultDenylist, cfg.Denylist)
	}

	// Expand ~ to $HOME
	for i, dir := range cfg.TargetDirs {
		if dir == "~" || (len(dir) > 0 && dir[0] == '~') {
			cfg.TargetDirs[i] = filepath.Join(os.Getenv("HOME"), dir[1:])
		}
	}

	return &cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
	if path == "" {
		path = ConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func mergeUnique(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))

	for _, item := range append(base, extra...) {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		merged = append(merged, item)
	}

	return merged
}
