package plugins

import (
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// PluginEnv provides dependencies and configuration to plugins during initialization and execution.
type PluginEnv struct {
	Repos  *repo.AllRepos
	Logger zerolog.Logger
	Config *config.Config
}

// NewPluginEnv creates a new PluginEnv instance.
func NewPluginEnv(repos *repo.AllRepos, logger zerolog.Logger, cfg *config.Config) *PluginEnv {
	return &PluginEnv{
		Repos:  repos,
		Logger: logger,
		Config: cfg,
	}
}
