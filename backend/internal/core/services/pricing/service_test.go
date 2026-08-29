package pricing_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/pricing"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
	"github.com/sysadminsmedia/homebox/backend/plugins/mtg"
)

type mockPricingProvider struct {
	id          string
	name        string
	providerID  string
	searchCalls int
}

func (m *mockPricingProvider) ID() string          { return m.id }
func (m *mockPricingProvider) Name() string        { return m.name }
func (m *mockPricingProvider) Description() string { return "Mock provider" }
func (m *mockPricingProvider) ProviderID() string  { return m.providerID }
func (m *mockPricingProvider) Init(ctx context.Context, env *plugins.PluginEnv) error {
	return nil
}
func (m *mockPricingProvider) SearchProducts(ctx context.Context, query string) ([]plugins.ProductSearchResult, error) {
	m.searchCalls++
	return []plugins.ProductSearchResult{
		{ProductID: "999", Name: "Mock Product", MarketPrice: 49.99, Provider: m.providerID},
	}, nil
}
func (m *mockPricingProvider) FetchPrice(ctx context.Context, productID string) (*plugins.PriceSnapshotResult, error) {
	return &plugins.PriceSnapshotResult{
		ProductID:   productID,
		MarketPrice: 50.0,
		Source:      m.providerID,
		RecordedAt:  time.Now(),
	}, nil
}
func (m *mockPricingProvider) DetectTrackingFromFields(fields []repo.EntityFieldData) (string, string, bool) {
	for _, f := range fields {
		if f.TextValue == "mock-tracking-url" {
			return "999", f.Name, true
		}
	}
	return "", "", false
}
func (m *mockPricingProvider) ExtractProductID(raw string) (string, bool) {
	if raw == "999" {
		return "999", true
	}
	return "", false
}

func TestPricingService_DetectTrackingFromFields(t *testing.T) {
	env := plugins.NewPluginEnv(nil, zerolog.Nop(), nil)
	reg := plugins.NewRegistry(env)

	mtgPlugin := mtg.NewPlugin()
	require.NoError(t, reg.Register(mtgPlugin))

	mockProvider := &mockPricingProvider{id: "mock", name: "Mock", providerID: "mock-prov"}
	require.NoError(t, reg.Register(mockProvider))

	svc := pricing.NewPricingService(nil, reg)

	// Test TCGPlayer detection
	fields := []repo.EntityFieldData{
		{Name: "Condition", TextValue: "Factory Sealed"},
		{Name: "TCG Link", TextValue: "https://www.tcgplayer.com/product/541164/magic-modern-horizons-3"},
	}

	pid, source, fieldName, ok := svc.DetectTrackingFromFields(fields)
	require.True(t, ok)
	assert.Equal(t, "541164", pid)
	assert.Equal(t, "tcgplayer", source)
	assert.Equal(t, "TCG Link", fieldName)

	// Test Mock provider detection
	mockFields := []repo.EntityFieldData{
		{Name: "Custom Field", TextValue: "mock-tracking-url"},
	}
	pid, source, fieldName, ok = svc.DetectTrackingFromFields(mockFields)
	require.True(t, ok)
	assert.Equal(t, "999", pid)
	assert.Equal(t, "mock-prov", source)
	assert.Equal(t, "Custom Field", fieldName)

	// Test non-matching
	noMatch := []repo.EntityFieldData{
		{Name: "Random", TextValue: "No match"},
	}
	pid, source, fieldName, ok = svc.DetectTrackingFromFields(noMatch)
	assert.False(t, ok)
	assert.Empty(t, pid)
	assert.Empty(t, source)
	assert.Empty(t, fieldName)
}

func TestPricingService_SearchProducts(t *testing.T) {
	env := plugins.NewPluginEnv(nil, zerolog.Nop(), nil)
	reg := plugins.NewRegistry(env)

	mockProvider := &mockPricingProvider{id: "mock", name: "Mock", providerID: "mock-prov"}
	require.NoError(t, reg.Register(mockProvider))

	svc := pricing.NewPricingService(nil, reg)

	results, err := svc.SearchProducts(context.Background(), "test")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "999", results[0].ProductID)
	assert.Equal(t, "Mock Product", results[0].Name)
}
