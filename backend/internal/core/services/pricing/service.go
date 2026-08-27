package pricing

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

var (
	tcgProductURLRegex = regexp.MustCompile(`(?i)tcgplayer\.com/(?:product/|magic/product/show\?id=)(\d+)`)
	pureNumberRegex    = regexp.MustCompile(`^\d{4,8}$`)
)

type PricingService struct {
	repos     *repo.AllRepos
	tcgClient *TCGCSVClient
}

func NewPricingService(repos *repo.AllRepos) *PricingService {
	return &PricingService{
		repos:     repos,
		tcgClient: NewTCGCSVClient(),
	}
}

func (s *PricingService) ExtractTCGProductID(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}

	// Match direct numeric ID
	if pureNumberRegex.MatchString(raw) {
		id, err := strconv.Atoi(raw)
		if err == nil && id > 0 {
			return id, true
		}
	}

	// Match URL patterns
	matches := tcgProductURLRegex.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		id, err := strconv.Atoi(matches[1])
		if err == nil && id > 0 {
			return id, true
		}
	}

	return 0, false
}

func (s *PricingService) DetectTCGPlayerLinkFromFields(fields []repo.EntityFieldData) (int, string, bool) {
	for _, f := range fields {
		if id, ok := s.ExtractTCGProductID(f.TextValue); ok {
			return id, f.Name, true
		}
	}
	return 0, "", false
}

func (s *PricingService) GetPriceHistory(ctx context.Context, gid, entityID uuid.UUID) ([]repo.PriceHistoryEntry, error) {
	return s.repos.PriceHistory.GetByEntity(ctx, gid, entityID)
}

func (s *PricingService) CreateManualPrice(ctx context.Context, gid, entityID uuid.UUID, data repo.PriceHistoryCreate) (repo.PriceHistoryEntry, error) {
	return s.repos.PriceHistory.Create(ctx, gid, entityID, data)
}

func (s *PricingService) DeletePrice(ctx context.Context, gid, entityID, priceID uuid.UUID) error {
	return s.repos.PriceHistory.Delete(ctx, gid, entityID, priceID)
}

func (s *PricingService) SyncEntityPrice(ctx context.Context, gid, entityID uuid.UUID) (*repo.PriceHistoryEntry, error) {
	entity, err := s.repos.Entities.GetOne(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %w", err)
	}

	productID := 0
	if entity.PriceTrackingID != "" {
		if id, ok := s.ExtractTCGProductID(entity.PriceTrackingID); ok {
			productID = id
		}
	}

	// Fallback to searching custom fields if tracking ID is not set
	if productID == 0 {
		if id, _, found := s.DetectTCGPlayerLinkFromFields(entity.Fields); found {
			productID = id
			// Enable tracking on the entity
			update := repo.EntityUpdate{
				ID:                       entity.ID,
				Name:                     entity.Name,
				Description:              entity.Description,
				SerialNumber:             entity.SerialNumber,
				ModelNumber:              entity.ModelNumber,
				Manufacturer:             entity.Manufacturer,
				Archived:                 entity.Archived,
				PurchaseFrom:             entity.PurchaseFrom,
				PurchasePrice:            entity.PurchasePrice,
				SoldTo:                   entity.SoldTo,
				SoldPrice:                entity.SoldPrice,
				SoldNotes:                entity.SoldNotes,
				Notes:                    entity.Notes,
				LifetimeWarranty:         entity.LifetimeWarranty,
				Insured:                  entity.Insured,
				WarrantyDetails:          entity.WarrantyDetails,
				Quantity:                 entity.Quantity,
				AssetID:                  entity.AssetID,
				SyncChildEntityLocations: entity.SyncChildEntityLocations,
				PriceTrackingEnabled:     true,
				PriceTrackingSource:      "tcgplayer",
				PriceTrackingID:          strconv.Itoa(id),
				TagIDs:                   nil,
			}
			if entity.EntityType != nil {
				update.EntityTypeID = entity.EntityType.ID
			}
			if entity.Parent != nil {
				update.ParentID = entity.Parent.ID
			}
			_, _ = s.repos.Entities.UpdateByGroup(ctx, gid, update)
		}
	}

	if productID == 0 {
		return nil, fmt.Errorf("no TCGPlayer product link or ID found for item %s", entity.Name)
	}

	priceRes, err := s.tcgClient.GetPrice(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("fetching price from TCGPlayer: %w", err)
	}

	notes := ""
	if priceRes.ProductName != "" {
		notes = priceRes.ProductName
		if priceRes.GroupName != "" {
			notes = fmt.Sprintf("%s (%s)", priceRes.ProductName, priceRes.GroupName)
		}
	}

	entry, err := s.repos.PriceHistory.RecordSnapshot(
		ctx,
		entityID,
		priceRes.MarketPrice,
		priceRes.LowPrice,
		priceRes.MidPrice,
		priceRes.HighPrice,
		priceRes.DirectLowPrice,
		"tcgplayer",
		strconv.Itoa(productID),
		priceRes.RecordedAt,
		notes,
	)
	if err != nil {
		return nil, fmt.Errorf("recording price snapshot: %w", err)
	}

	return &entry, nil
}

func (s *PricingService) AutoDetectEntityTracking(ctx context.Context, gid, entityID uuid.UUID) (*repo.EntityOut, bool, error) {
	entity, err := s.repos.Entities.GetOne(ctx, entityID)
	if err != nil {
		return nil, false, err
	}

	productID, _, found := s.DetectTCGPlayerLinkFromFields(entity.Fields)
	if !found {
		return &entity, false, nil
	}

	update := repo.EntityUpdate{
		ID:                       entity.ID,
		Name:                     entity.Name,
		Description:              entity.Description,
		SerialNumber:             entity.SerialNumber,
		ModelNumber:              entity.ModelNumber,
		Manufacturer:             entity.Manufacturer,
		Archived:                 entity.Archived,
		PurchaseFrom:             entity.PurchaseFrom,
		PurchasePrice:            entity.PurchasePrice,
		SoldTo:                   entity.SoldTo,
		SoldPrice:                entity.SoldPrice,
		SoldNotes:                entity.SoldNotes,
		Notes:                    entity.Notes,
		LifetimeWarranty:         entity.LifetimeWarranty,
		Insured:                  entity.Insured,
		WarrantyDetails:          entity.WarrantyDetails,
		Quantity:                 entity.Quantity,
		AssetID:                  entity.AssetID,
		SyncChildEntityLocations: entity.SyncChildEntityLocations,
		PriceTrackingEnabled:     true,
		PriceTrackingSource:      "tcgplayer",
		PriceTrackingID:          strconv.Itoa(productID),
		TagIDs:                   nil,
	}
	if entity.EntityType != nil {
		update.EntityTypeID = entity.EntityType.ID
	}
	if entity.Parent != nil {
		update.ParentID = entity.Parent.ID
	}

	updated, err := s.repos.Entities.UpdateByGroup(ctx, gid, update)
	if err != nil {
		return nil, false, err
	}

	// Trigger initial price fetch asynchronously or synchronously
	_, _ = s.SyncEntityPrice(ctx, gid, entityID)
	refreshed, err := s.repos.Entities.GetOne(ctx, entityID)
	if err == nil {
		return &refreshed, true, nil
	}

	return &updated, true, nil
}

func (s *PricingService) syncEntitiesList(ctx context.Context, entities []*ent.Entity) (int, error) {
	updated := 0
	for _, e := range entities {
		productID := 0
		if e.PriceTrackingID != "" {
			if id, ok := s.ExtractTCGProductID(e.PriceTrackingID); ok {
				productID = id
			}
		}

		if productID == 0 && len(e.Edges.Fields) > 0 {
			fields := make([]repo.EntityFieldData, len(e.Edges.Fields))
			for i, f := range e.Edges.Fields {
				fields[i] = repo.EntityFieldData{
					Name:      f.Name,
					TextValue: f.TextValue,
				}
			}
			if id, _, found := s.DetectTCGPlayerLinkFromFields(fields); found {
				productID = id
			}
		}

		if productID == 0 {
			continue
		}

		priceRes, err := s.tcgClient.GetPrice(ctx, productID)
		if err != nil {
			log.Warn().Err(err).Str("entity_id", e.ID.String()).Int("product_id", productID).Msg("failed to fetch price for tracked entity")
			continue
		}

		notes := ""
		if priceRes.ProductName != "" {
			notes = priceRes.ProductName
			if priceRes.GroupName != "" {
				notes = fmt.Sprintf("%s (%s)", priceRes.ProductName, priceRes.GroupName)
			}
		}

		_, err = s.repos.PriceHistory.RecordSnapshot(
			ctx,
			e.ID,
			priceRes.MarketPrice,
			priceRes.LowPrice,
			priceRes.MidPrice,
			priceRes.HighPrice,
			priceRes.DirectLowPrice,
			"tcgplayer",
			strconv.Itoa(productID),
			priceRes.RecordedAt,
			notes,
		)
		if err != nil {
			log.Warn().Err(err).Str("entity_id", e.ID.String()).Msg("failed to record price snapshot")
			continue
		}

		updated++
	}

	return updated, nil
}

func (s *PricingService) SyncAllTrackedEntities(ctx context.Context) (int, error) {
	entities, err := s.repos.PriceHistory.GetTrackedEntities(ctx)
	if err != nil {
		return 0, err
	}
	return s.syncEntitiesList(ctx, entities)
}

func (s *PricingService) SyncGroupTrackedEntities(ctx context.Context, gid uuid.UUID) (int, error) {
	entities, err := s.repos.PriceHistory.GetTrackedEntitiesByGroup(ctx, gid)
	if err != nil {
		return 0, err
	}
	return s.syncEntitiesList(ctx, entities)
}

func (s *PricingService) SyncEntitiesBulk(ctx context.Context, gid uuid.UUID, entityIDs []uuid.UUID) (int, error) {
	if len(entityIDs) == 0 {
		return 0, nil
	}
	entities, err := s.repos.PriceHistory.GetTrackedEntitiesByIDs(ctx, gid, entityIDs)
	if err != nil {
		return 0, err
	}
	return s.syncEntitiesList(ctx, entities)
}

func (s *PricingService) SearchProducts(ctx context.Context, query string) ([]ProductSearchResult, error) {
	return s.tcgClient.SearchProducts(ctx, query)
}
