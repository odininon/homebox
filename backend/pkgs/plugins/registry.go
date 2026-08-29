package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

// Registry manages registered plugins, capability lookups, and lifecycle hooks.
type Registry struct {
	mu               sync.RWMutex
	env              *PluginEnv
	plugins          map[string]Plugin
	pricingProviders map[string]PricingProvider
}

// NewRegistry instantiates a new plugin registry with the given environment.
func NewRegistry(env *PluginEnv) *Registry {
	return &Registry{
		env:              env,
		plugins:          make(map[string]Plugin),
		pricingProviders: make(map[string]PricingProvider),
	}
}

// Register registers and initializes a plugin into the registry.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := p.ID()
	if id == "" {
		return fmt.Errorf("plugin ID cannot be empty")
	}

	if _, exists := r.plugins[id]; exists {
		return fmt.Errorf("plugin with ID %q is already registered", id)
	}

	if r.env != nil {
		if err := p.Init(context.Background(), r.env); err != nil {
			return fmt.Errorf("initializing plugin %q: %w", id, err)
		}
	}

	r.plugins[id] = p

	if pp, ok := p.(PricingProvider); ok {
		r.pricingProviders[pp.ProviderID()] = pp
	}

	if r.env != nil {
		r.env.Logger.Info().Str("plugin_id", id).Str("plugin_name", p.Name()).Msg("registered plugin")
	}

	return nil
}

// GetPlugin retrieves a registered plugin by ID.
func (r *Registry) GetPlugin(id string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[id]
	return p, ok
}

// GetPricingProvider retrieves a registered pricing provider by provider ID (e.g. "tcgplayer").
func (r *Registry) GetPricingProvider(providerID string) (PricingProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pp, ok := r.pricingProviders[providerID]
	return pp, ok
}

// AllPricingProviders returns all registered pricing providers.
func (r *Registry) AllPricingProviders() []PricingProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]PricingProvider, 0, len(r.pricingProviders))
	for _, pp := range r.pricingProviders {
		list = append(list, pp)
	}
	return list
}

// AllPlugins returns all registered plugins.
func (r *Registry) AllPlugins() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		list = append(list, p)
	}
	return list
}

// GetPluginInfos returns metadata about all registered plugins for public API responses.
func (r *Registry) GetPluginInfos() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(r.plugins))
	for _, p := range r.plugins {
		caps := []string{}
		if _, ok := p.(PricingProvider); ok {
			caps = append(caps, "pricing")
		}
		if _, ok := p.(RouteRegistrar); ok {
			caps = append(caps, "routes")
		}
		if _, ok := p.(ScheduledJobProvider); ok {
			caps = append(caps, "scheduled_jobs")
		}
		if _, ok := p.(EntityHookProvider); ok {
			caps = append(caps, "entity_hooks")
		}

		infos = append(infos, PluginInfo{
			ID:           p.ID(),
			Name:         p.Name(),
			Description:  p.Description(),
			Capabilities: caps,
		})
	}
	return infos
}

// MountRoutes mounts all plugin HTTP routes onto the specified Chi router.
// Each plugin that implements RouteRegistrar gets mounted at /plugins/{plugin_id}.
func (r *Registry) MountRoutes(router chi.Router) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if rr, ok := p.(RouteRegistrar); ok {
			router.Route("/plugins/"+p.ID(), func(sub chi.Router) {
				rr.RegisterRoutes(sub, r.env)
			})
		}
	}
}

// GetScheduledJobs collects all scheduled recurring jobs from registered plugins.
func (r *Registry) GetScheduledJobs() []ScheduledJob {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var jobs []ScheduledJob
	for _, p := range r.plugins {
		if sjp, ok := p.(ScheduledJobProvider); ok {
			jobs = append(jobs, sjp.ScheduledJobs(r.env)...)
		}
	}
	return jobs
}

// EmitEntitySaved invokes OnEntitySaved hooks across all registered plugins.
func (r *Registry) EmitEntitySaved(ctx context.Context, entity *repo.EntityOut) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if eh, ok := p.(EntityHookProvider); ok {
			if err := eh.OnEntitySaved(ctx, r.env, entity); err != nil && r.env != nil {
				r.env.Logger.Warn().Err(err).Str("plugin_id", p.ID()).Str("entity_id", entity.ID.String()).Msg("plugin OnEntitySaved hook error")
			}
		}
	}
}

// EmitEntityDeleted invokes OnEntityDeleted hooks across all registered plugins.
func (r *Registry) EmitEntityDeleted(ctx context.Context, entityID uuid.UUID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if eh, ok := p.(EntityHookProvider); ok {
			if err := eh.OnEntityDeleted(ctx, r.env, entityID); err != nil && r.env != nil {
				r.env.Logger.Warn().Err(err).Str("plugin_id", p.ID()).Str("entity_id", entityID.String()).Msg("plugin OnEntityDeleted hook error")
			}
		}
	}
}
