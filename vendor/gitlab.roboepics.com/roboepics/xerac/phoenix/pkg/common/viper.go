package common

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func SetupViper(defaults map[string]any) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("PHX")
	viper.AutomaticEnv()

	viper.SetConfigType("yaml")
	viper.AddConfigPath(
		filepath.Join(homeDir, "/.phoenix"))
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}
