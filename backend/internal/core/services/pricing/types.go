package pricing

import "time"

type PriceResult struct {
	ProductID      int       `json:"productId"`
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
}

type ProductSearchResult struct {
	ProductID   int     `json:"productId"`
	Name        string  `json:"name"`
	CleanName   string  `json:"cleanName"`
	GroupName   string  `json:"groupName"`
	GroupID     int     `json:"groupId"`
	MarketPrice float64 `json:"marketPrice"`
	ImageURL    string  `json:"imageUrl"`
	URL         string  `json:"url"`
}
