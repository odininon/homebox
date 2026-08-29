package plugins_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
)

type mockPlugin struct {
	id          string
	name        string
	initialized bool
	savedCalled bool
}

func (m *mockPlugin) ID() string          { return m.id }
func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Description() string { return "Mock plugin description" }
func (m *mockPlugin) Init(ctx context.Context, env *plugins.PluginEnv) error {
	m.initialized = true
	return nil
}

type mockPricingPlugin struct {
	mockPlugin
}

func (m *mockPricingPlugin) ProviderID() string { return "mock-pricing" }
func (m *mockPricingPlugin) SearchProducts(ctx context.Context, query string) ([]plugins.ProductSearchResult, error) {
	return []plugins.ProductSearchResult{
		{ProductID: "123", Name: "Mock Product", MarketPrice: 99.5},
	}, nil
}
func (m *mockPricingPlugin) FetchPrice(ctx context.Context, productID string) (*plugins.PriceSnapshotResult, error) {
	return &plugins.PriceSnapshotResult{
		ProductID:   productID,
		MarketPrice: 100.0,
		Source:      "mock-pricing",
		RecordedAt:  time.Now(),
	}, nil
}
func (m *mockPricingPlugin) DetectTrackingFromFields(fields []repo.EntityFieldData) (string, string, bool) {
	for _, f := range fields {
		if f.TextValue == "track-me" {
			return "123", f.Name, true
		}
	}
	return "", "", false
}
func (m *mockPricingPlugin) ExtractProductID(raw string) (string, bool) {
	if raw == "123" {
		return "123", true
	}
	return "", false
}

func (m *mockPricingPlugin) RegisterRoutes(r chi.Router, env *plugins.PluginEnv) {
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
}

func (m *mockPricingPlugin) ScheduledJobs(env *plugins.PluginEnv) []plugins.ScheduledJob {
	return []plugins.ScheduledJob{
		{
			Name:     "mock-job",
			Interval: time.Hour,
			Run:      func(ctx context.Context) {},
		},
	}
}

func (m *mockPricingPlugin) OnEntitySaved(ctx context.Context, env *plugins.PluginEnv, entity *repo.EntityOut) error {
	m.savedCalled = true
	return nil
}

func (m *mockPricingPlugin) OnEntityDeleted(ctx context.Context, env *plugins.PluginEnv, entityID uuid.UUID) error {
	return nil
}

func TestPluginRegistry(t *testing.T) {
	env := plugins.NewPluginEnv(nil, zerolog.Nop(), nil)
	reg := plugins.NewRegistry(env)

	mock := &mockPricingPlugin{
		mockPlugin: mockPlugin{id: "mock", name: "Mock Plugin"},
	}

	err := reg.Register(mock)
	require.NoError(t, err)
	assert.True(t, mock.initialized)

	// Duplicate registration error
	err = reg.Register(mock)
	require.Error(t, err)

	// Lookup plugin
	p, ok := reg.GetPlugin("mock")
	assert.True(t, ok)
	assert.Equal(t, "Mock Plugin", p.Name())

	// Lookup pricing provider
	pp, ok := reg.GetPricingProvider("mock-pricing")
	assert.True(t, ok)
	assert.Equal(t, "mock-pricing", pp.ProviderID())

	allPP := reg.AllPricingProviders()
	assert.Len(t, allPP, 1)

	// Plugin info
	infos := reg.GetPluginInfos()
	require.Len(t, infos, 1)
	assert.Equal(t, "mock", infos[0].ID)
	assert.Contains(t, infos[0].Capabilities, "pricing")
	assert.Contains(t, infos[0].Capabilities, "routes")
	assert.Contains(t, infos[0].Capabilities, "scheduled_jobs")
	assert.Contains(t, infos[0].Capabilities, "entity_hooks")

	// Routes
	router := chi.NewRouter()
	reg.MountRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/plugins/mock/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())

	// Scheduled jobs
	jobs := reg.GetScheduledJobs()
	require.Len(t, jobs, 1)
	assert.Equal(t, "mock-job", jobs[0].Name)

	// Entity hook
	entity := &repo.EntityOut{
		EntitySummary: repo.EntitySummary{
			ID:   uuid.New(),
			Name: "Test Entity",
		},
	}
	reg.EmitEntitySaved(context.Background(), entity)
	assert.True(t, mock.savedCalled)
}
