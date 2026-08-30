package pricing

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
)

type PricingService struct {
	repos   *repo.AllRepos
	plugins *plugins.Registry
}

func NewPricingService(repos *repo.AllRepos, registry *plugins.Registry) *PricingService {
	return &PricingService{
		repos:   repos,
		plugins: registry,
	}
}

// DetectTrackingFromFields iterates over all registered pricing providers to detect a trackable product ID/URL.
func (s *PricingService) DetectTrackingFromFields(fields []repo.EntityFieldData) (productID string, source string, fieldName string, ok bool) {
	if s.plugins == nil {
		return "", "", "", false
	}

	for _, provider := range s.plugins.AllPricingProviders() {
		if pid, fname, found := provider.DetectTrackingFromFields(fields); found {
			return pid, provider.ProviderID(), fname, true
		}
	}
	return "", "", "", false
}

// DetectTracking detects a trackable product ID/URL from custom fields or from the item name matching catalog products.
func (s *PricingService) DetectTracking(ctx context.Context, name string, fields []repo.EntityFieldData) (productID string, source string, fieldName string, ok bool) {
	if s.plugins == nil {
		return "", "", "", false
	}

	// 1. Custom fields detection (TCGPlayer URL or numeric ID)
	if pid, src, fname, found := s.DetectTrackingFromFields(fields); found {
		return pid, src, fname, true
	}

	// 2. Catalog product name match
	trimmedName := strings.TrimSpace(name)
	if trimmedName != "" {
		for _, provider := range s.plugins.AllPricingProviders() {
			results, err := provider.SearchProducts(ctx, trimmedName)
			if err == nil && len(results) > 0 {
				cleanName := strings.ToLower(trimmedName)
				for _, r := range results {
					if strings.EqualFold(strings.TrimSpace(r.Name), trimmedName) || strings.EqualFold(strings.TrimSpace(r.CleanName), cleanName) {
						return r.ProductID, provider.ProviderID(), "name", true
					}
				}
				// Check for " - " split e.g. "Set Name - Product Name"
				parts := strings.Split(trimmedName, " - ")
				if len(parts) > 1 {
					subName := strings.TrimSpace(parts[0] + " " + parts[1])
					for _, r := range results {
						if strings.EqualFold(strings.TrimSpace(r.Name), subName) || strings.EqualFold(strings.TrimSpace(r.CleanName), strings.ToLower(subName)) {
							return r.ProductID, provider.ProviderID(), "name", true
						}
					}
				}
				// If 1 result or top result is a strong match
				if len(results) == 1 {
					return results[0].ProductID, provider.ProviderID(), "name", true
				}
				top := results[0]
				if strings.Contains(cleanName, strings.ToLower(top.Name)) || strings.Contains(strings.ToLower(top.Name), cleanName) {
					return top.ProductID, provider.ProviderID(), "name", true
				}
			}
		}
	}

	return "", "", "", false
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

	source := entity.PriceTrackingSource
	productID := entity.PriceTrackingID

	// Fallback to auto-detection from custom fields or name if not set
	if productID == "" || source == "" {
		if pid, src, _, found := s.DetectTracking(ctx, entity.Name, entity.Fields); found {
			productID = pid
			source = src

			// Enable tracking on entity
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
				PriceTrackingSource:      source,
				PriceTrackingID:          productID,
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

	if productID == "" || source == "" {
		return nil, fmt.Errorf("no pricing provider or product ID configured for item %s", entity.Name)
	}

	if s.plugins == nil {
		return nil, fmt.Errorf("plugin registry not initialized")
	}

	provider, ok := s.plugins.GetPricingProvider(source)
	if !ok {
		return nil, fmt.Errorf("unknown pricing provider %q", source)
	}

	priceRes, err := provider.FetchPrice(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("fetching price from provider %q: %w", source, err)
	}

	entry, err := s.repos.PriceHistory.RecordSnapshot(
		ctx,
		entityID,
		priceRes.MarketPrice,
		priceRes.LowPrice,
		priceRes.MidPrice,
		priceRes.HighPrice,
		priceRes.DirectLowPrice,
		source,
		productID,
		priceRes.RecordedAt,
		priceRes.Notes,
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

	productID, source, _, found := s.DetectTracking(ctx, entity.Name, entity.Fields)
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
		PriceTrackingSource:      source,
		PriceTrackingID:          productID,
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

	// Trigger initial price fetch
	_, _ = s.SyncEntityPrice(ctx, gid, entityID)
	refreshed, err := s.repos.Entities.GetOne(ctx, entityID)
	if err == nil {
		return &refreshed, true, nil
	}

	return &updated, true, nil
}

func (s *PricingService) syncEntitiesList(ctx context.Context, entities []*ent.Entity) (int, error) {
	if s.plugins == nil {
		return 0, nil
	}

	updated := 0
	for _, e := range entities {
		source := e.PriceTrackingSource
		productID := e.PriceTrackingID

		if source == "" || productID == "" {
			var fields []repo.EntityFieldData
			if len(e.Edges.Fields) > 0 {
				fields = make([]repo.EntityFieldData, len(e.Edges.Fields))
				for i, f := range e.Edges.Fields {
					fields[i] = repo.EntityFieldData{
						Name:      f.Name,
						TextValue: f.TextValue,
					}
				}
			}
			if pid, src, _, found := s.DetectTracking(ctx, e.Name, fields); found {
				productID = pid
				source = src
			}
		}

		if source == "" || productID == "" {
			continue
		}

		provider, ok := s.plugins.GetPricingProvider(source)
		if !ok {
			log.Warn().Str("entity_id", e.ID.String()).Str("source", source).Msg("unknown pricing provider for entity")
			continue
		}

		priceRes, err := provider.FetchPrice(ctx, productID)
		if err != nil {
			log.Warn().Err(err).Str("entity_id", e.ID.String()).Str("product_id", productID).Str("source", source).Msg("failed to fetch price for tracked entity")
			continue
		}

		_, err = s.repos.PriceHistory.RecordSnapshot(
			ctx,
			e.ID,
			priceRes.MarketPrice,
			priceRes.LowPrice,
			priceRes.MidPrice,
			priceRes.HighPrice,
			priceRes.DirectLowPrice,
			source,
			productID,
			priceRes.RecordedAt,
			priceRes.Notes,
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
	entities, err := s.repos.PriceHistory.GetAllItemEntitiesByGroup(ctx, gid)
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

func (s *PricingService) SearchProducts(ctx context.Context, query string, providerIDs ...string) ([]ProductSearchResult, error) {
	if s.plugins == nil {
		return nil, nil
	}

	var providers []plugins.PricingProvider
	if len(providerIDs) > 0 && providerIDs[0] != "" {
		if p, ok := s.plugins.GetPricingProvider(providerIDs[0]); ok {
			providers = append(providers, p)
		}
	} else {
		providers = s.plugins.AllPricingProviders()
	}

	var allResults []ProductSearchResult
	for _, p := range providers {
		res, err := p.SearchProducts(ctx, query)
		if err != nil {
			log.Warn().Err(err).Str("provider", p.ProviderID()).Msg("product search error")
			continue
		}
		allResults = append(allResults, res...)
	}

	return allResults, nil
}
