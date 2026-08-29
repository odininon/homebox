package plugins

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

// Plugin is the base interface implemented by all Homebox plugins.
type Plugin interface {
	ID() string
	Name() string
	Description() string
	Init(ctx context.Context, env *PluginEnv) error
}

// ProductSearchResult represents a single search match from a pricing or catalog provider.
type ProductSearchResult struct {
	ProductID   string  `json:"productId"`
	Name        string  `json:"name"`
	CleanName   string  `json:"cleanName"`
	GroupName   string  `json:"groupName,omitempty"`
	GroupID     int     `json:"groupId,omitempty"`
	MarketPrice float64 `json:"marketPrice"`
	ImageURL    string  `json:"imageUrl,omitempty"`
	URL         string  `json:"url,omitempty"`
	Provider    string  `json:"provider,omitempty"`
}

// PriceSnapshotResult represents live market price data retrieved by a pricing provider.
type PriceSnapshotResult struct {
	ProductID      string    `json:"productId"`
	ProductName    string    `json:"productName,omitempty"`
	GroupName      string    `json:"groupName,omitempty"`
	MarketPrice    float64   `json:"marketPrice"`
	LowPrice       float64   `json:"lowPrice"`
	MidPrice       float64   `json:"midPrice"`
	HighPrice      float64   `json:"highPrice"`
	DirectLowPrice float64   `json:"directLowPrice"`
	ImageURL       string    `json:"imageUrl,omitempty"`
	Source         string    `json:"source"`
	RecordedAt     time.Time `json:"recordedAt"`
	Notes          string    `json:"notes,omitempty"`
}

// PricingProvider is an interface for plugins that provide product search, pricing lookups,
// and custom field detection for item valuation.
type PricingProvider interface {
	Plugin
	ProviderID() string
	SearchProducts(ctx context.Context, query string) ([]ProductSearchResult, error)
	FetchPrice(ctx context.Context, productID string) (*PriceSnapshotResult, error)
	DetectTrackingFromFields(fields []repo.EntityFieldData) (productID string, fieldName string, ok bool)
	ExtractProductID(raw string) (productID string, ok bool)
}

// RouteRegistrar is implemented by plugins that expose their own HTTP API endpoints.
// These routes will be mounted under /api/v1/plugins/{plugin_id}.
type RouteRegistrar interface {
	RegisterRoutes(r chi.Router, env *PluginEnv)
}

// ScheduledJob defines a background recurring task provided by a plugin.
type ScheduledJob struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context)
}

// ScheduledJobProvider is implemented by plugins that need to run background recurring tasks.
type ScheduledJobProvider interface {
	ScheduledJobs(env *PluginEnv) []ScheduledJob
}

// EntityHookProvider is implemented by plugins that need to react to entity lifecycle events.
type EntityHookProvider interface {
	OnEntitySaved(ctx context.Context, env *PluginEnv, entity *repo.EntityOut) error
	OnEntityDeleted(ctx context.Context, env *PluginEnv, entityID uuid.UUID) error
}

// PluginInfo contains public metadata about an installed and initialized plugin.
type PluginInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}
