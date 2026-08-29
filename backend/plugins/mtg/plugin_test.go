package mtg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/plugins/mtg"
)

func TestMTGPlugin_Metadata(t *testing.T) {
	p := mtg.NewPlugin()
	assert.Equal(t, "mtg", p.ID())
	assert.Equal(t, "tcgplayer", p.ProviderID())
	assert.NotEmpty(t, p.Name())
	assert.NotEmpty(t, p.Description())
}

func TestMTGPlugin_ExtractProductID(t *testing.T) {
	p := mtg.NewPlugin()

	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"541234", "541234", true},
		{"https://www.tcgplayer.com/product/541234/magic-modern-horizons-3", "541234", true},
		{"https://tcgplayer.com/magic/product/show?id=12345", "12345", true},
		{"invalid text", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		got, ok := p.ExtractProductID(tt.input)
		assert.Equal(t, tt.ok, ok, "input: %s", tt.input)
		if ok {
			assert.Equal(t, tt.expected, got)
		}
	}
}

func TestMTGPlugin_DetectTrackingFromFields(t *testing.T) {
	p := mtg.NewPlugin()

	fields := []repo.EntityFieldData{
		{Name: "Color", TextValue: "Blue"},
		{Name: "TCG Link", TextValue: "https://www.tcgplayer.com/product/543210/magic-the-gathering"},
	}

	id, fieldName, ok := p.DetectTrackingFromFields(fields)
	require.True(t, ok)
	assert.Equal(t, "543210", id)
	assert.Equal(t, "TCG Link", fieldName)
}

func TestMTGPlugin_Init(t *testing.T) {
	p := mtg.NewPlugin()
	err := p.Init(context.Background(), nil)
	assert.NoError(t, err)
}
