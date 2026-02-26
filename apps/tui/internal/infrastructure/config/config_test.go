package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigNormalizedClampsAndDefaults(t *testing.T) {
	cfg := Config{
		DataDir:              "",
		APIBaseURL:           "",
		ThemePack:            "unknown",
		KeyHintsDensity:      "LOUD",
		StartupMode:          "else",
		HighlightStyle:       "flash",
		ContentWidth:         10,
		MarginHorizontal:     -5,
		LineSpacing:          0,
		ParagraphSpacing:     7,
		SpreadThreshold:      0,
		MinSpreadWidth:       999,
		PrimaryOverrideColor: "",
	}

	n := cfg.Normalized()
	d := Default()

	if n.DataDir == "" {
		t.Fatalf("expected data dir default")
	}
	if n.APIBaseURL == "" {
		t.Fatalf("expected API base URL default")
	}
	if n.ThemePack != d.ThemePack {
		t.Fatalf("expected theme fallback %q, got %q", d.ThemePack, n.ThemePack)
	}
	if n.KeyHintsDensity != d.KeyHintsDensity {
		t.Fatalf("expected key hints fallback %q, got %q", d.KeyHintsDensity, n.KeyHintsDensity)
	}
	if n.StartupMode != d.StartupMode {
		t.Fatalf("expected startup fallback %q, got %q", d.StartupMode, n.StartupMode)
	}
	if n.HighlightStyle != d.HighlightStyle {
		t.Fatalf("expected highlight fallback %q, got %q", d.HighlightStyle, n.HighlightStyle)
	}
	if n.ContentWidth != 40 {
		t.Fatalf("expected clamped content width 40, got %d", n.ContentWidth)
	}
	if n.MarginHorizontal != 0 {
		t.Fatalf("expected clamped margin 0, got %d", n.MarginHorizontal)
	}
	if n.LineSpacing != d.LineSpacing {
		t.Fatalf("expected default line spacing %d, got %d", d.LineSpacing, n.LineSpacing)
	}
	if n.ParagraphSpacing != 2 {
		t.Fatalf("expected clamped paragraph spacing 2, got %d", n.ParagraphSpacing)
	}
	if n.SpreadThreshold != 220 {
		t.Fatalf("expected clamped spread threshold 220, got %d", n.SpreadThreshold)
	}
	if n.MinSpreadWidth != n.SpreadThreshold {
		t.Fatalf("expected min spread width mirror spread threshold")
	}
	if n.PrimaryOverrideColor == "" {
		t.Fatalf("expected primary override color default")
	}
}

func TestSaveAndLoadFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := Default()
	cfg.DataDir = filepath.Join(dir, "data")
	cfg.APIBaseURL = "http://localhost:9090"
	cfg.ThemePack = ThemePackNord
	cfg.KeyHintsDensity = KeyHintsDensityCompact
	cfg.StartupMode = StartupModeLibrary
	cfg.HighlightStyle = HighlightStyleBlock
	cfg.PrimaryOverrideEnabled = true
	cfg.PrimaryOverrideColor = "117"
	cfg.ContentWidth = 132
	cfg.MarginHorizontal = 4
	cfg.LineSpacing = 2
	cfg.ParagraphSpacing = 1
	cfg.SpreadThreshold = 144
	cfg.MinSpreadWidth = 144
	cfg.DeleteConfirmation = false

	if err := Save(path, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.ThemePack != ThemePackNord {
		t.Fatalf("expected theme pack nord, got %q", loaded.ThemePack)
	}
	if loaded.APIBaseURL != "http://localhost:9090" {
		t.Fatalf("expected API base URL round-trip, got %q", loaded.APIBaseURL)
	}
	if loaded.KeyHintsDensity != KeyHintsDensityCompact {
		t.Fatalf("expected compact hints, got %q", loaded.KeyHintsDensity)
	}
	if loaded.StartupMode != StartupModeLibrary {
		t.Fatalf("expected startup library, got %q", loaded.StartupMode)
	}
	if loaded.HighlightStyle != HighlightStyleBlock {
		t.Fatalf("expected block highlight, got %q", loaded.HighlightStyle)
	}
	if !loaded.PrimaryOverrideEnabled || loaded.PrimaryOverrideColor != "117" {
		t.Fatalf("expected primary override to round-trip")
	}
	if loaded.ContentWidth != 132 || loaded.MarginHorizontal != 4 {
		t.Fatalf("expected width/margin round-trip, got %d/%d", loaded.ContentWidth, loaded.MarginHorizontal)
	}
	if loaded.LineSpacing != 2 || loaded.ParagraphSpacing != 1 {
		t.Fatalf("expected spacing round-trip, got %d/%d", loaded.LineSpacing, loaded.ParagraphSpacing)
	}
	if loaded.SpreadThreshold != 144 || loaded.MinSpreadWidth != 144 {
		t.Fatalf("expected spread threshold round-trip, got %d/%d", loaded.SpreadThreshold, loaded.MinSpreadWidth)
	}
	if loaded.DeleteConfirmation {
		t.Fatalf("expected delete confirmation false round-trip")
	}
}

func TestMergeSettingsKeepsBaseDataDir(t *testing.T) {
	base := Default()
	base.DataDir = "/base"
	base.APIBaseURL = "http://localhost:8080"
	base.ThemePack = ThemePackDefault

	imported := Default()
	imported.DataDir = "/imported"
	imported.APIBaseURL = "http://localhost:7070"
	imported.ThemePack = ThemePackDracula
	imported.StartupMode = StartupModeLibrary

	merged := MergeSettings(base, imported)
	if merged.DataDir != "/base" {
		t.Fatalf("expected base data dir preserved, got %q", merged.DataDir)
	}
	if merged.ThemePack != ThemePackDracula {
		t.Fatalf("expected imported theme pack, got %q", merged.ThemePack)
	}
	if merged.APIBaseURL != "http://localhost:7070" {
		t.Fatalf("expected imported API base URL, got %q", merged.APIBaseURL)
	}
	if merged.StartupMode != StartupModeLibrary {
		t.Fatalf("expected imported startup mode, got %q", merged.StartupMode)
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.toml")
	if err := Save(path, Default()); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}
