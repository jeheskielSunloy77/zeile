package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	StartupModeResume  = "resume"
	StartupModeLibrary = "library"

	KeyHintsDensityFull    = "full"
	KeyHintsDensityCompact = "compact"
	KeyHintsDensityHidden  = "hidden"

	ThemePackDefault    = "default"
	ThemePackDracula    = "dracula"
	ThemePackGruvbox    = "gruvbox"
	ThemePackNord       = "nord"
	ThemePackCatppuccin = "catppuccin"

	HighlightStyleUnderline = "underline"
	HighlightStyleReverse   = "reverse"
	HighlightStyleBlock     = "block"
)

type Config struct {
	DataDir            string `toml:"data_dir"`
	ManagedCopyDefault bool   `toml:"managed_copy_default"`

	// Deprecated alias preserved for backward compatibility.
	MinSpreadWidth int `toml:"min_spread_width"`

	ThemePack              string `toml:"theme_pack"`
	PrimaryOverrideEnabled bool   `toml:"primary_override_enabled"`
	PrimaryOverrideColor   string `toml:"primary_override_color"`

	ContentWidth       int  `toml:"content_width"`
	MarginHorizontal   int  `toml:"margin_horizontal"`
	LineSpacing        int  `toml:"line_spacing"`
	ParagraphSpacing   int  `toml:"paragraph_spacing"`
	SpreadThreshold    int  `toml:"spread_threshold"`
	HighContrast       bool `toml:"high_contrast"`
	DeleteConfirmation bool `toml:"delete_confirmation"`

	StartupMode     string `toml:"startup_mode"`
	KeyHintsDensity string `toml:"key_hints_density"`
	HighlightStyle  string `toml:"highlight_style"`
}

type Loaded struct {
	Config Config
	Path   string
}

func Load() (Loaded, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return Loaded{}, err
	}

	cfg := Default()

	if _, statErr := os.Stat(configPath); statErr == nil {
		if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
			return Loaded{}, fmt.Errorf("decode config: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return Loaded{}, fmt.Errorf("stat config: %w", statErr)
	} else {
		if err := writeDefaults(configPath, cfg); err != nil {
			return Loaded{}, err
		}
	}

	cfg = cfg.Normalized()
	return Loaded{Config: cfg, Path: configPath}, nil
}

func LoadFile(path string) (Config, error) {
	cfg := Default()
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg.Normalized(), nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(cfg.Normalized()); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func MergeSettings(base, imported Config) Config {
	base.ManagedCopyDefault = imported.ManagedCopyDefault
	base.ThemePack = imported.ThemePack
	base.PrimaryOverrideEnabled = imported.PrimaryOverrideEnabled
	base.PrimaryOverrideColor = imported.PrimaryOverrideColor
	base.ContentWidth = imported.ContentWidth
	base.MarginHorizontal = imported.MarginHorizontal
	base.LineSpacing = imported.LineSpacing
	base.ParagraphSpacing = imported.ParagraphSpacing
	base.SpreadThreshold = imported.SpreadThreshold
	base.MinSpreadWidth = imported.SpreadThreshold
	base.HighContrast = imported.HighContrast
	base.DeleteConfirmation = imported.DeleteConfirmation
	base.StartupMode = imported.StartupMode
	base.KeyHintsDensity = imported.KeyHintsDensity
	base.HighlightStyle = imported.HighlightStyle
	return base.Normalized()
}

func (cfg Config) Normalized() Config {
	d := Default()

	if strings.TrimSpace(cfg.DataDir) == "" {
		cfg.DataDir = d.DataDir
	}

	if cfg.SpreadThreshold <= 0 {
		if cfg.MinSpreadWidth > 0 {
			cfg.SpreadThreshold = cfg.MinSpreadWidth
		} else {
			cfg.SpreadThreshold = d.SpreadThreshold
		}
	}
	cfg.SpreadThreshold = clamp(cfg.SpreadThreshold, 80, 220)
	cfg.MinSpreadWidth = cfg.SpreadThreshold

	if cfg.ContentWidth <= 0 {
		cfg.ContentWidth = d.ContentWidth
	}
	cfg.ContentWidth = clamp(cfg.ContentWidth, 40, 240)
	cfg.MarginHorizontal = clamp(cfg.MarginHorizontal, 0, 20)
	if cfg.LineSpacing <= 0 {
		cfg.LineSpacing = d.LineSpacing
	}
	cfg.LineSpacing = clamp(cfg.LineSpacing, 1, 3)
	if cfg.ParagraphSpacing < 0 {
		cfg.ParagraphSpacing = d.ParagraphSpacing
	}
	cfg.ParagraphSpacing = clamp(cfg.ParagraphSpacing, 0, 2)

	cfg.ThemePack = normalizeEnum(cfg.ThemePack, d.ThemePack, map[string]struct{}{
		ThemePackDefault:    {},
		ThemePackDracula:    {},
		ThemePackGruvbox:    {},
		ThemePackNord:       {},
		ThemePackCatppuccin: {},
	})
	cfg.KeyHintsDensity = normalizeEnum(cfg.KeyHintsDensity, d.KeyHintsDensity, map[string]struct{}{
		KeyHintsDensityFull:    {},
		KeyHintsDensityCompact: {},
		KeyHintsDensityHidden:  {},
	})
	cfg.StartupMode = normalizeEnum(cfg.StartupMode, d.StartupMode, map[string]struct{}{
		StartupModeResume:  {},
		StartupModeLibrary: {},
	})
	cfg.HighlightStyle = normalizeEnum(cfg.HighlightStyle, d.HighlightStyle, map[string]struct{}{
		HighlightStyleUnderline: {},
		HighlightStyleReverse:   {},
		HighlightStyleBlock:     {},
	})

	if strings.TrimSpace(cfg.PrimaryOverrideColor) == "" {
		cfg.PrimaryOverrideColor = d.PrimaryOverrideColor
	}

	return cfg
}

func Default() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return Config{
		DataDir:              filepath.Join(homeDir, ".zeile"),
		ManagedCopyDefault:   true,
		MinSpreadWidth:       120,
		ThemePack:            ThemePackDefault,
		PrimaryOverrideColor: "205",
		ContentWidth:         120,
		MarginHorizontal:     2,
		LineSpacing:          1,
		ParagraphSpacing:     0,
		SpreadThreshold:      120,
		DeleteConfirmation:   true,
		StartupMode:          StartupModeResume,
		KeyHintsDensity:      KeyHintsDensityFull,
		HighlightStyle:       HighlightStyleReverse,
	}
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "zeile", "config.toml"), nil
}

func writeDefaults(path string, cfg Config) error {
	return Save(path, cfg)
}

func normalizeEnum(value, fallback string, allowed map[string]struct{}) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowed[normalized]; !ok {
		return fallback
	}
	return normalized
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
