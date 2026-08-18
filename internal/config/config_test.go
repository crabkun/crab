package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMaster(t *testing.T) {
	t.Parallel()
	cfg, err := File{
		Mode:      "master",
		LogLevel:  "info",
		ListenAt:  "0.0.0.0:51324",
		MasterKey: "secret",
	}.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != ModeMaster || cfg.Master.ListenAt == "" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestValidateRejectsUnknownCipher(t *testing.T) {
	t.Parallel()
	_, err := File{
		Mode:     "client",
		LogLevel: "debug",
		Master:   "127.0.0.1:1",
		Ports: []Port{{
			LocalAddress:   "127.0.0.1:2",
			PortKey:        "k",
			EncryptMethod:  "rc4",
			CompressMethod: "null",
		}},
	}.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	const body = `{
		"mode": "master",
		"log_level": "error",
		"listen_at": "127.0.0.1:0",
		"master_key": "k"
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Master.MasterKey != "k" {
		t.Fatalf("master key = %q", cfg.Master.MasterKey)
	}
}
