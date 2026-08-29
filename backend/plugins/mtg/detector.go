package mtg

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

var (
	tcgProductURLRegex = regexp.MustCompile(`(?i)tcgplayer\.com/(?:product/|magic/product/show\?id=)(\d+)`)
	pureNumberRegex    = regexp.MustCompile(`^\d{4,8}$`)
)

// ExtractTCGProductID extracts a numeric TCGPlayer product ID from a raw ID or URL string.
func ExtractTCGProductID(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	// Match direct numeric ID
	if pureNumberRegex.MatchString(raw) {
		id, err := strconv.Atoi(raw)
		if err == nil && id > 0 {
			return raw, true
		}
	}

	// Match URL patterns
	matches := tcgProductURLRegex.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		id, err := strconv.Atoi(matches[1])
		if err == nil && id > 0 {
			return matches[1], true
		}
	}

	return "", false
}

// DetectTCGPlayerLinkFromFields scans an entity's custom fields for a TCGPlayer product URL or ID.
func DetectTCGPlayerLinkFromFields(fields []repo.EntityFieldData) (string, string, bool) {
	for _, f := range fields {
		if id, ok := ExtractTCGProductID(f.TextValue); ok {
			return id, f.Name, true
		}
	}
	return "", "", false
}
