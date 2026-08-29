package mtg

import (
	"context"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
)

// Ensure Plugin satisfies the required interfaces at compile time.
var (
	_ plugins.Plugin          = (*Plugin)(nil)
	_ plugins.PricingProvider = (*Plugin)(nil)
	_ plugins.RouteRegistrar  = (*Plugin)(nil)
)

// Plugin represents the Magic: The Gathering / TCGPlayer sealed product pricing integration.
type Plugin struct {
	client *TCGCSVClient
	env    *plugins.PluginEnv
}

// NewPlugin instantiates a new MTG Plugin.
func NewPlugin() *Plugin {
	return &Plugin{
		client: NewTCGCSVClient(),
	}
}

// ID returns the unique identifier for the plugin.
func (p *Plugin) ID() string {
	return "mtg"
}

// Name returns the human-readable name of the plugin.
func (p *Plugin) Name() string {
	return "Magic: The Gathering Sealed Product Tracker"
}

// Description returns a brief explanation of what the plugin does.
func (p *Plugin) Description() string {
	return "Live market pricing, sealed product search, and historical valuation powered by TCGPlayer."
}

// Init initializes the plugin with the provided environment.
func (p *Plugin) Init(ctx context.Context, env *plugins.PluginEnv) error {
	p.env = env
	return nil
}

// ProviderID matches the source string stored on entities and price history entries (e.g. "tcgplayer").
func (p *Plugin) ProviderID() string {
	return "tcgplayer"
}

// SearchProducts queries the TCGPlayer catalog for sealed MTG products.
func (p *Plugin) SearchProducts(ctx context.Context, query string) ([]plugins.ProductSearchResult, error) {
	return p.client.SearchProducts(ctx, query)
}

// FetchPrice retrieves the current market price snapshot for a given TCGPlayer product ID.
func (p *Plugin) FetchPrice(ctx context.Context, productID string) (*plugins.PriceSnapshotResult, error) {
	return p.client.GetPrice(ctx, productID)
}

// DetectTrackingFromFields checks if any custom fields on an entity contain a TCGPlayer product URL or ID.
func (p *Plugin) DetectTrackingFromFields(fields []repo.EntityFieldData) (string, string, bool) {
	return DetectTCGPlayerLinkFromFields(fields)
}

// ExtractProductID parses a numeric product ID from a raw string or URL.
func (p *Plugin) ExtractProductID(raw string) (string, bool) {
	return ExtractTCGProductID(raw)
}
