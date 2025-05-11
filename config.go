package main

import (
	"github.com/desxz/go-message-scheduler/client"
	"github.com/spf13/viper"
)

type Config struct {
	Worker        WorkerConfig
	WebhookClient client.WebhookClientConfig
	Cache         CacheConfig
	Pool          PoolConfig
}

func NewConfig(configPath, configEnv string) (*Config, error) {
	viperConfig, err := readConfig(configPath, configEnv)
	if err != nil {
		return nil, err
	}

	config := new(Config)

	if err := viperConfig.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

func readConfig(configPath, configName string) (*viper.Viper, error) {
	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName(configName)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return v, nil
}
