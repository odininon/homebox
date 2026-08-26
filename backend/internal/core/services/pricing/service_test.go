package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

func TestExtractTCGProductID(t *testing.T) {
	svc := &PricingService{}

	tests := []struct {
		input    string
		expected int
		valid    bool
	}{
		{
			input:    "https://www.tcgplayer.com/product/541164/magic-modern-horizons-3-modern-horizons-3-play-booster-display",
			expected: 541164,
			valid:    true,
		},
		{
			input:    "https://tcgplayer.com/product/12345",
			expected: 12345,
			valid:    true,
		},
		{
			input:    "http://shop.tcgplayer.com/magic/product/show?id=98765",
			expected: 98765,
			valid:    true,
		},
		{
			input:    "541164",
			expected: 541164,
			valid:    true,
		},
		{
			input:    "  541164  ",
			expected: 541164,
			valid:    true,
		},
		{
			input:    "https://www.ebay.com/itm/123456",
			expected: 0,
			valid:    false,
		},
		{
			input:    "",
			expected: 0,
			valid:    false,
		},
	}

	for _, tt := range tests {
		id, ok := svc.ExtractTCGProductID(tt.input)
		assert.Equal(t, tt.valid, ok, "Input: %s", tt.input)
		if tt.valid {
			assert.Equal(t, tt.expected, id, "Input: %s", tt.input)
		}
	}
}

func TestDetectTCGPlayerLinkFromFields(t *testing.T) {
	svc := &PricingService{}

	fields := []repo.EntityFieldData{
		{
			Name:      "Condition",
			TextValue: "Factory Sealed",
		},
		{
			Name:      "TCGPlayer",
			TextValue: "https://www.tcgplayer.com/product/541164/magic-modern-horizons-3-modern-horizons-3-play-booster-display",
		},
		{
			Name:      "Location Bin",
			TextValue: "Shelf 4A",
		},
	}

	id, fieldName, found := svc.DetectTCGPlayerLinkFromFields(fields)
	require.True(t, found)
	assert.Equal(t, 541164, id)
	assert.Equal(t, "TCGPlayer", fieldName)

	// Test no matching field
	noMatchFields := []repo.EntityFieldData{
		{
			Name:      "Condition",
			TextValue: "Factory Sealed",
		},
	}
	_, _, found = svc.DetectTCGPlayerLinkFromFields(noMatchFields)
	assert.False(t, found)
}

func TestTCGCSVClient_GetPrice(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/groups":
			_ = json.NewEncoder(w).Encode(tcgResponse[TCGGroup]{
				Success: true,
				Results: []TCGGroup{
					{
						GroupID:     23444,
						Name:        "Modern Horizons 3",
						PublishedOn: "2024-06-14T00:00:00",
					},
				},
			})
		case "/23444/prices":
			marketPrice := 294.67
			lowPrice := 294.45
			midPrice := 349.99
			highPrice := 604.19
			_ = json.NewEncoder(w).Encode(tcgResponse[TCGPrice]{
				Success: true,
				Results: []TCGPrice{
					{
						ProductID:   541164,
						MarketPrice: &marketPrice,
						LowPrice:    &lowPrice,
						MidPrice:    &midPrice,
						HighPrice:   &highPrice,
						SubTypeName: "Normal",
					},
				},
			})
		case "/23444/products":
			_ = json.NewEncoder(w).Encode(tcgResponse[TCGProduct]{
				Success: true,
				Results: []TCGProduct{
					{
						ProductID: 541164,
						Name:      "Modern Horizons 3 - Play Booster Display",
						CleanName: "Modern Horizons 3 Play Booster Display",
						GroupID:   23444,
						URL:       "https://www.tcgplayer.com/product/541164",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	client := NewTCGCSVClient()
	client.baseURL = mockServer.URL

	ctx := context.Background()
	res, err := client.GetPrice(ctx, 541164)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, 541164, res.ProductID)
	assert.InDelta(t, 294.67, res.MarketPrice, 0.001)
	assert.InDelta(t, 294.45, res.LowPrice, 0.001)
	assert.InDelta(t, 349.99, res.MidPrice, 0.001)
	assert.InDelta(t, 604.19, res.HighPrice, 0.001)
	assert.Equal(t, "Modern Horizons 3 - Play Booster Display", res.ProductName)
	assert.Equal(t, "Modern Horizons 3", res.GroupName)
	assert.Equal(t, "tcgplayer", res.Source)
}
