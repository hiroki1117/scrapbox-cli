package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// File は設定ファイル (JSON) の内容である。認証情報を含むため、パーミッション 0600 で保存する。
type File struct {
	Project     string      `json:"project,omitempty"`
	Credentials Credentials `json:"credentials,omitempty"`
}

// FilePath は設定ファイルのパスを返す。
// SB_CONFIG > $XDG_CONFIG_HOME/sb/config.json > ~/.config/sb/config.json の順に決める。
func FilePath(getenv func(string) string) (string, error) {
	if p := getenv(EnvConfigFile); p != "" {
		return p, nil
	}
	if dir := getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "sb", "config.json"), nil
	}
	home := getenv("HOME")
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
	}
	return filepath.Join(home, ".config", "sb", "config.json"), nil
}

// ReadFile は設定ファイルを読み込む。ファイルが存在しない場合はゼロ値を返す。
func ReadFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config file: %w", err)
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return f, nil
}

// WriteFile は設定ファイルを書き込む。ディレクトリは 0700、ファイルは 0600 で作成する。
// 書き込み途中で壊れないよう、一時ファイルに書いてから rename する。
func WriteFile(path string, f File) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("write config file: %w", err)
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// UpdateFile は設定ファイルを読み込み、fn で変更して書き戻す。
func UpdateFile(path string, fn func(*File)) error {
	f, err := ReadFile(path)
	if err != nil {
		return err
	}
	fn(&f)
	return WriteFile(path, f)
}
