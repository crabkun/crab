package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads a JSON config file (same schema as the original Crab) and
// applies optional CRAB_* environment overrides for top-level fields.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	v.SetEnvPrefix("CRAB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("load configure file failed: %w", err)
	}

	var file File
	if err := v.Unmarshal(&file); err != nil {
		return nil, fmt.Errorf("unmarshal configure file failed: %w", err)
	}

	cfg, err := file.Validate()
	if err != nil {
		return nil, fmt.Errorf("validate configure file failed: %w", err)
	}
	return cfg, nil
}
