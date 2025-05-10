package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_New(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		configEnv  string
		want       *Config
		wantErr    bool
	}{
		{
			name:       "should return config when valid config file is provided",
			configPath: ".config",
			configEnv:  "dev",
			want: &Config{
				Worker: WorkerConfig{
					WorkerJobTimeout:  10 * time.Second,
					WorkerJobInterval: 1 * time.Second,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.configPath, tt.configEnv)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
