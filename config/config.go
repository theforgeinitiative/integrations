package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/config")
	viper.AddConfigPath("/etc/forgebot")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("fatal error config file: %w", err)
	}
	return nil
}
