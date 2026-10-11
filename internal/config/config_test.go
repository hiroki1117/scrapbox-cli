package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad_Precedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := WriteFile(path, File{Project: "fileproj", Credentials: Credentials{PersonalAccessToken: "filetoken"}}); err != nil {
		t.Fatal(err)
	}

	t.Run("file only", func(t *testing.T) {
		cfg, err := Load(envFunc(map[string]string{EnvConfigFile: path}))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Project != "fileproj" || cfg.CredentialSource != SourceFile {
			t.Errorf("unexpected config: %+v", cfg)
		}
		if typ, v := cfg.Credentials.Active(); typ != AuthPersonalAccessToken || v != "filetoken" {
			t.Errorf("active = %s %s", typ, v)
		}
	})

	t.Run("env overrides file without mixing credentials", func(t *testing.T) {
		cfg, err := Load(envFunc(map[string]string{
			EnvConfigFile: path,
			EnvProject:    "envproj",
			EnvSessionID:  "envsid",
		}))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Project != "envproj" || cfg.CredentialSource != SourceEnv {
			t.Errorf("unexpected config: %+v", cfg)
		}
		// 設定ファイルの PAT が優先されず、環境変数の SID が使われること。
		if typ, v := cfg.Credentials.Active(); typ != AuthSessionID || v != "envsid" {
			t.Errorf("active = %s %s", typ, v)
		}
	})
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load(envFunc(map[string]string{EnvConfigFile: filepath.Join(t.TempDir(), "none.json")}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CredentialSource != SourceNone || cfg.Project != "" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestWriteFile_Permission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sb", "config.json")
	if err := UpdateFile(path, func(f *File) { f.Credentials = NewCredentials(AuthServiceAccount, "key") }); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows はパーミッションビットを持たないため検証しない。
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
	f, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Credentials.ServiceAccountKey != "key" {
		t.Errorf("credentials = %+v", f.Credentials)
	}
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{EnvConfigFile: "/x/c.json", "HOME": "/h"}, "/x/c.json"},
		{map[string]string{"XDG_CONFIG_HOME": "/xdg", "HOME": "/h"}, filepath.Join("/xdg", "sb", "config.json")},
		{map[string]string{"HOME": "/h"}, filepath.Join("/h", ".config", "sb", "config.json")},
	}
	for _, tt := range tests {
		got, err := FilePath(envFunc(tt.env))
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("FilePath(%v) = %q, want %q", tt.env, got, tt.want)
		}
	}
}
