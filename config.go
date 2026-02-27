package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Buffer   BufferConfig    `yaml:"buffer"`
	Channels []ChannelConfig `yaml:"channels"`
	Post     PostConfig      `yaml:"post"`
}

type BufferConfig struct {
	Token string `yaml:"token"`
}

type ChannelConfig struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	CharLimit int    `yaml:"char_limit"`
}

type PostConfig struct {
	CharLimit int    `yaml:"char_limit"`
	Template  string `yaml:"template"`
}

const localConfigFile = "hugo-buffer.yaml"

func loadConfig() (*Config, error) {
	path, err := findConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Blog-level override: hugo-buffer.yaml in cwd
	if fileExists(localConfigFile) {
		local, err := os.ReadFile(localConfigFile)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", localConfigFile, err)
		}
		var localCfg Config
		if err := yaml.Unmarshal(local, &localCfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", localConfigFile, err)
		}
		if localCfg.Post.Template != "" {
			cfg.Post.Template = localCfg.Post.Template
		}
	}

	// Environment variable overrides token
	if tok := os.Getenv("BUFFER_TOKEN"); tok != "" {
		cfg.Buffer.Token = tok
	}

	return &cfg, nil
}

func findConfigPath() (string, error) {
	// 1. Explicit env var
	if p := os.Getenv("HUGO_BUFFER_CONFIG"); p != "" {
		return p, nil
	}

	// 2. XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		p := filepath.Join(xdg, "hugo-buffer", "config.yaml")
		if fileExists(p) {
			return p, nil
		}
	}

	// 3. Default ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	p := filepath.Join(home, ".config", "hugo-buffer", "config.yaml")
	if fileExists(p) {
		return p, nil
	}

	return "", fmt.Errorf("config file not found; create ~/.config/hugo-buffer/config.yaml (see config.yaml.example)")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
