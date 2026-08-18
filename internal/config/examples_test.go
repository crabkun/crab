package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestExampleConfigsStillLoad(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "config-examples")

	cases := []struct {
		file string
		mode string
	}{
		{file: "config_master.json", mode: ModeMaster},
		{file: "config_server.json", mode: ModeServer},
		{file: "config_client.json", mode: ModeClient},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(filepath.Join(root, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Mode != tc.mode {
				t.Fatalf("mode = %q, want %q", cfg.Mode, tc.mode)
			}
		})
	}
}
