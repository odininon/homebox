package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	tcgcsvBaseURL   = "https://tcgcsv.com/tcgplayer/1"
	defaultTimeout  = 15 * time.Second
	cacheTTL        = 12 * time.Hour
	userAgentHeader = "Homebox/1.0 (MTG Sealed Price Tracker; https://github.com/sysadminsmedia/homebox)"
)

type TCGCSVClient struct {
	httpClient *http.Client
	baseURL    string

	mu           sync.RWMutex
	groups       []TCGGroup
	groupsExpiry time.Time

	groupPrices       map[int][]TCGPrice
	groupPricesExpiry map[int]time.Time

	groupProducts       map[int][]TCGProduct
	groupProductsExpiry map[int]time.Time

	productToGroup map[int]int
}

func NewTCGCSVClient() *TCGCSVClient {
	return &TCGCSVClient{
		httpClient:          &http.Client{Timeout: defaultTimeout},
		baseURL:             tcgcsvBaseURL,
		groupPrices:         make(map[int][]TCGPrice),
		groupPricesExpiry:   make(map[int]time.Time),
		groupProducts:       make(map[int][]TCGProduct),
		groupProductsExpiry: make(map[int]time.Time),
		productToGroup:      make(map[int]int),
	}
}

type TCGGroup struct {
	GroupID        int    `json:"groupId"`
	Name           string `json:"name"`
	Abbreviation   string `json:"abbreviation"`
	IsSupplemental bool   `json:"isSupplemental"`
	PublishedOn    string `json:"publishedOn"`
	ModifiedOn     string `json:"modifiedOn"`
	CategoryID     int    `json:"categoryId"`
}

type TCGPrice struct {
	ProductID      int      `json:"productId"`
	LowPrice       *float64 `json:"lowPrice"`
	MidPrice       *float64 `json:"midPrice"`
	HighPrice      *float64 `json:"highPrice"`
	MarketPrice    *float64 `json:"marketPrice"`
	DirectLowPrice *float64 `json:"directLowPrice"`
	SubTypeName    string   `json:"subTypeName"`
}

type TCGProduct struct {
	ProductID  int    `json:"productId"`
	Name       string `json:"name"`
	CleanName  string `json:"cleanName"`
	ImageURL   string `json:"imageUrl"`
	CategoryID int    `json:"categoryId"`
	GroupID    int    `json:"groupId"`
	URL        string `json:"url"`
}

type tcgResponse[T any] struct {
	TotalItems int      `json:"totalItems"`
	Success    bool     `json:"success"`
	Errors     []string `json:"errors"`
	Results    []T      `json:"results"`
}

func (c *TCGCSVClient) fetch(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgentHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tcgcsv request to %s failed with status %d", endpoint, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}

	return json.Unmarshal(bodyBytes, target)
}

func (c *TCGCSVClient) GetGroups(ctx context.Context) ([]TCGGroup, error) {
	c.mu.RLock()
	if len(c.groups) > 0 && time.Now().Before(c.groupsExpiry) {
		groups := c.groups
		c.mu.RUnlock()
		return groups, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.groups) > 0 && time.Now().Before(c.groupsExpiry) {
		return c.groups, nil
	}

	var res tcgResponse[TCGGroup]
	if err := c.fetch(ctx, "/groups", &res); err != nil {
		return nil, fmt.Errorf("fetching groups: %w", err)
	}

	// Sort groups with newest published on top
	sort.Slice(res.Results, func(i, j int) bool {
		return res.Results[i].PublishedOn > res.Results[j].PublishedOn
	})

	c.groups = res.Results
	c.groupsExpiry = time.Now().Add(cacheTTL)
	return c.groups, nil
}

func (c *TCGCSVClient) GetGroupPrices(ctx context.Context, groupID int) ([]TCGPrice, error) {
	c.mu.RLock()
	if prices, ok := c.groupPrices[groupID]; ok && time.Now().Before(c.groupPricesExpiry[groupID]) {
		c.mu.RUnlock()
		return prices, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if prices, ok := c.groupPrices[groupID]; ok && time.Now().Before(c.groupPricesExpiry[groupID]) {
		return prices, nil
	}

	var res tcgResponse[TCGPrice]
	endpoint := fmt.Sprintf("/%d/prices", groupID)
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, fmt.Errorf("fetching group prices for %d: %w", groupID, err)
	}

	for _, p := range res.Results {
		c.productToGroup[p.ProductID] = groupID
	}

	c.groupPrices[groupID] = res.Results
	c.groupPricesExpiry[groupID] = time.Now().Add(cacheTTL)
	return res.Results, nil
}

func (c *TCGCSVClient) GetGroupProducts(ctx context.Context, groupID int) ([]TCGProduct, error) {
	c.mu.RLock()
	if products, ok := c.groupProducts[groupID]; ok && time.Now().Before(c.groupProductsExpiry[groupID]) {
		c.mu.RUnlock()
		return products, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if products, ok := c.groupProducts[groupID]; ok && time.Now().Before(c.groupProductsExpiry[groupID]) {
		return products, nil
	}

	var res tcgResponse[TCGProduct]
	endpoint := fmt.Sprintf("/%d/products", groupID)
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, fmt.Errorf("fetching group products for %d: %w", groupID, err)
	}

	for _, p := range res.Results {
		c.productToGroup[p.ProductID] = groupID
	}

	c.groupProducts[groupID] = res.Results
	c.groupProductsExpiry[groupID] = time.Now().Add(cacheTTL)
	return res.Results, nil
}

func (c *TCGCSVClient) FindProductGroup(ctx context.Context, productID int) (int, error) {
	c.mu.RLock()
	if gid, ok := c.productToGroup[productID]; ok {
		c.mu.RUnlock()
		return gid, nil
	}
	c.mu.RUnlock()

	groups, err := c.GetGroups(ctx)
	if err != nil {
		return 0, err
	}

	// Search groups in parallel batches to find the product
	batchSize := 10
	for i := 0; i < len(groups); i += batchSize {
		end := i + batchSize
		if end > len(groups) {
			end = len(groups)
		}

		type result struct {
			groupID int
			found   bool
		}
		resChan := make(chan result, end-i)
		var wg sync.WaitGroup

		for _, grp := range groups[i:end] {
			wg.Add(1)
			go func(gid int) {
				defer wg.Done()
				prices, err := c.GetGroupPrices(ctx, gid)
				if err != nil {
					return
				}
				for _, p := range prices {
					if p.ProductID == productID {
						resChan <- result{groupID: gid, found: true}
						return
					}
				}
			}(grp.GroupID)
		}

		wg.Wait()
		close(resChan)

		for r := range resChan {
			if r.found {
				return r.groupID, nil
			}
		}
	}

	return 0, fmt.Errorf("product ID %d not found in MTG catalog", productID)
}

func valOrZero(p *float64) float64 {
	if p == nil {
		return 0.0
	}
	return *p
}

func (c *TCGCSVClient) GetPrice(ctx context.Context, productID int) (*PriceResult, error) {
	groupID, err := c.FindProductGroup(ctx, productID)
	if err != nil {
		return nil, err
	}

	prices, err := c.GetGroupPrices(ctx, groupID)
	if err != nil {
		return nil, err
	}

	var foundPrice *TCGPrice
	for _, p := range prices {
		if p.ProductID == productID {
			foundPrice = &p
			break
		}
	}

	if foundPrice == nil {
		return nil, fmt.Errorf("price not found for product ID %d in group %d", productID, groupID)
	}

	marketPrice := valOrZero(foundPrice.MarketPrice)
	if marketPrice == 0 {
		marketPrice = valOrZero(foundPrice.MidPrice)
	}
	if marketPrice == 0 {
		marketPrice = valOrZero(foundPrice.LowPrice)
	}

	// Try to attach product name / group name if available
	productName := ""
	groupName := ""
	imageURL := ""

	groups, _ := c.GetGroups(ctx)
	for _, g := range groups {
		if g.GroupID == groupID {
			groupName = g.Name
			break
		}
	}

	products, _ := c.GetGroupProducts(ctx, groupID)
	for _, prod := range products {
		if prod.ProductID == productID {
			productName = prod.Name
			imageURL = prod.ImageURL
			break
		}
	}

	return &PriceResult{
		ProductID:      productID,
		ProductName:    productName,
		GroupName:      groupName,
		MarketPrice:    marketPrice,
		LowPrice:       valOrZero(foundPrice.LowPrice),
		MidPrice:       valOrZero(foundPrice.MidPrice),
		HighPrice:      valOrZero(foundPrice.HighPrice),
		DirectLowPrice: valOrZero(foundPrice.DirectLowPrice),
		ImageURL:       imageURL,
		Source:         "tcgplayer",
		RecordedAt:     time.Now(),
	}, nil
}

func isSealedProductName(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{
		"booster", "box", "bundle", "display", "deck", "pack", "case",
		"prerelease", "fat pack", "collector", "commander", "jumpstart",
		"gift", "starter", "draft", "set booster", "play booster", "blister",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func (c *TCGCSVClient) SearchProducts(ctx context.Context, query string) ([]ProductSearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, nil
	}

	groups, err := c.GetGroups(ctx)
	if err != nil {
		return nil, err
	}

	queryWords := strings.Fields(query)

	// Score and sort groups: prioritize groups whose names match query words
	type scoredGroup struct {
		group TCGGroup
		score int
	}

	var scoredGroups []scoredGroup
	for _, grp := range groups {
		gName := strings.ToLower(grp.Name)
		score := 0
		if strings.Contains(gName, query) {
			score += 100
		}
		for _, w := range queryWords {
			if strings.Contains(gName, w) {
				score += 10
			}
		}
		scoredGroups = append(scoredGroups, scoredGroup{group: grp, score: score})
	}

	sort.Slice(scoredGroups, func(i, j int) bool {
		if scoredGroups[i].score != scoredGroups[j].score {
			return scoredGroups[i].score > scoredGroups[j].score
		}
		return scoredGroups[i].group.PublishedOn > scoredGroups[j].group.PublishedOn
	})

	// Search candidate groups (up to 40 candidate groups)
	maxCandidateGroups := 40
	if len(scoredGroups) > maxCandidateGroups {
		scoredGroups = scoredGroups[:maxCandidateGroups]
	}

	type groupResult struct {
		items []ProductSearchResult
	}

	resChan := make(chan groupResult, len(scoredGroups))
	sem := make(chan struct{}, 8) // Limit concurrency to 8 workers
	var wg sync.WaitGroup

	for _, sg := range scoredGroups {
		wg.Add(1)
		go func(grp TCGGroup) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			products, err := c.GetGroupProducts(ctx, grp.GroupID)
			if err != nil {
				return
			}

			prices, _ := c.GetGroupPrices(ctx, grp.GroupID)
			priceMap := make(map[int]float64, len(prices))
			for _, pr := range prices {
				priceMap[pr.ProductID] = valOrZero(pr.MarketPrice)
			}

			var matches []ProductSearchResult
			for _, prod := range products {
				pName := strings.ToLower(prod.Name)
				pClean := strings.ToLower(prod.CleanName)

				allMatch := true
				for _, w := range queryWords {
					if !strings.Contains(pName, w) && !strings.Contains(pClean, w) && !strings.Contains(strings.ToLower(grp.Name), w) {
						allMatch = false
						break
					}
				}

				if allMatch || strings.Contains(pName, query) || strings.Contains(pClean, query) {
					matches = append(matches, ProductSearchResult{
						ProductID:   prod.ProductID,
						Name:        prod.Name,
						CleanName:   prod.CleanName,
						GroupName:   grp.Name,
						GroupID:     grp.GroupID,
						MarketPrice: priceMap[prod.ProductID],
						ImageURL:    prod.ImageURL,
						URL:         prod.URL,
					})
				}
			}

			if len(matches) > 0 {
				resChan <- groupResult{items: matches}
			}
		}(sg.group)
	}

	wg.Wait()
	close(resChan)

	var allResults []ProductSearchResult
	for gr := range resChan {
		allResults = append(allResults, gr.items...)
	}

	// Sort results: prioritize sealed products, then products matching query closest
	sort.Slice(allResults, func(i, j int) bool {
		iSealed := isSealedProductName(allResults[i].Name)
		jSealed := isSealedProductName(allResults[j].Name)
		if iSealed != jSealed {
			return iSealed
		}
		return allResults[i].MarketPrice > allResults[j].MarketPrice
	})

	maxResults := 50
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	return allResults, nil
}
