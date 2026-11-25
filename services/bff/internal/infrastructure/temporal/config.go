package temporal

import (
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HostPort                       string        `envconfig:"HOST_PORT" default:"localhost:7233"`
	Namespace                      string        `envconfig:"NAMESPACE" default:"default"`
	TaskQueue                      string        `envconfig:"TASK_QUEUE" default:"account-opening"`
	WorkflowExecutionTimeout       time.Duration `envconfig:"WORKFLOW_EXECUTION_TIMEOUT" default:"24h"`
	ActivityStartToCloseTimeout    time.Duration `envconfig:"ACTIVITY_START_TO_CLOSE_TIMEOUT" default:"30s"`
	ActivityScheduleToCloseTimeout time.Duration `envconfig:"ACTIVITY_SCHEDULE_TO_CLOSE_TIMEOUT" default:"5m"`
	ActivityMaximumAttempts        int32         `envconfig:"ACTIVITY_MAXIMUM_ATTEMPTS" default:"5"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("TEMPORAL", &cfg); err != nil {
		return Config{}, fmt.Errorf("processing temporal config: %w", err)
	}
	if cfg.TaskQueue == "" {
		return Config{}, errors.New("temporal task queue is required")
	}
	if cfg.WorkflowExecutionTimeout <= 0 {
		return Config{}, errors.New("temporal workflow execution timeout must be positive")
	}
	if cfg.ActivityStartToCloseTimeout <= 0 {
		return Config{}, errors.New("temporal activity start-to-close timeout must be positive")
	}
	if cfg.ActivityScheduleToCloseTimeout <= 0 {
		return Config{}, errors.New("temporal activity schedule-to-close timeout must be positive")
	}
	if cfg.ActivityMaximumAttempts <= 0 {
		return Config{}, errors.New("temporal activity maximum attempts must be positive")
	}
	return cfg, nil
}
