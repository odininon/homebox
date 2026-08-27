package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/entity"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/entitypricehistory"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/group"
)

type PriceHistoryRepository struct {
	db *ent.Client
}

func NewPriceHistoryRepository(db *ent.Client) *PriceHistoryRepository {
	return &PriceHistoryRepository{db: db}
}

type PriceHistoryCreate struct {
	Price      float64   `json:"price"`
	MarketLow  float64   `json:"marketLow"`
	MarketMid  float64   `json:"marketMid"`
	MarketHigh float64   `json:"marketHigh"`
	DirectLow  float64   `json:"directLow"`
	Source     string    `json:"source"`
	SourceID   string    `json:"sourceId"`
	RecordedAt time.Time `json:"recordedAt"`
	Notes      string    `json:"notes"`
}

type PriceHistoryEntry struct {
	ID         uuid.UUID `json:"id"`
	EntityID   uuid.UUID `json:"entityId"`
	Price      float64   `json:"price"`
	MarketLow  float64   `json:"marketLow"`
	MarketMid  float64   `json:"marketMid"`
	MarketHigh float64   `json:"marketHigh"`
	DirectLow  float64   `json:"directLow"`
	Source     string    `json:"source"`
	SourceID   string    `json:"sourceId"`
	RecordedAt time.Time `json:"recordedAt"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"createdAt"`
}

func mapPriceHistoryEntry(e *ent.EntityPriceHistory) PriceHistoryEntry {
	return PriceHistoryEntry{
		ID:         e.ID,
		EntityID:   e.EntityID,
		Price:      e.Price,
		MarketLow:  e.MarketLow,
		MarketMid:  e.MarketMid,
		MarketHigh: e.MarketHigh,
		DirectLow:  e.DirectLow,
		Source:     e.Source,
		SourceID:   e.SourceID,
		RecordedAt: e.RecordedAt,
		Notes:      e.Notes,
		CreatedAt:  e.CreatedAt,
	}
}

func (r *PriceHistoryRepository) GetByEntity(ctx context.Context, gid, entityID uuid.UUID) ([]PriceHistoryEntry, error) {
	entries, err := r.db.EntityPriceHistory.Query().
		Where(
			entitypricehistory.EntityIDEQ(entityID),
			entitypricehistory.HasEntityWith(
				entity.HasGroupWith(group.ID(gid)),
			),
		).
		Order(ent.Asc(entitypricehistory.FieldRecordedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]PriceHistoryEntry, len(entries))
	for i, e := range entries {
		result[i] = mapPriceHistoryEntry(e)
	}
	return result, nil
}

func (r *PriceHistoryRepository) Create(ctx context.Context, gid, entityID uuid.UUID, data PriceHistoryCreate) (PriceHistoryEntry, error) {
	if err := assertEntityInGroup(ctx, r.db.Entity, gid, entityID); err != nil {
		return PriceHistoryEntry{}, err
	}

	recordedAt := data.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}

	source := data.Source
	if source == "" {
		source = "manual"
	}

	entry, err := r.db.EntityPriceHistory.Create().
		SetEntityID(entityID).
		SetPrice(data.Price).
		SetMarketLow(data.MarketLow).
		SetMarketMid(data.MarketMid).
		SetMarketHigh(data.MarketHigh).
		SetDirectLow(data.DirectLow).
		SetSource(source).
		SetSourceID(data.SourceID).
		SetRecordedAt(recordedAt).
		SetNotes(data.Notes).
		Save(ctx)
	if err != nil {
		return PriceHistoryEntry{}, err
	}

	// Update entity current market price
	if data.Price > 0 {
		_ = r.db.Entity.UpdateOneID(entityID).
			SetCurrentMarketPrice(data.Price).
			SetLastPriceSyncAt(recordedAt).
			Exec(ctx)
	}

	return mapPriceHistoryEntry(entry), nil
}

func (r *PriceHistoryRepository) Delete(ctx context.Context, gid, entityID, priceID uuid.UUID) error {
	if err := assertEntityInGroup(ctx, r.db.Entity, gid, entityID); err != nil {
		return err
	}

	_, err := r.db.EntityPriceHistory.Delete().
		Where(
			entitypricehistory.IDEQ(priceID),
			entitypricehistory.EntityIDEQ(entityID),
		).
		Exec(ctx)
	return err
}

func (r *PriceHistoryRepository) RecordSnapshot(ctx context.Context, entityID uuid.UUID, price, low, mid, high, directLow float64, source, sourceID string, recordedAt time.Time, notes string) (PriceHistoryEntry, error) {
	if recordedAt.IsZero() {
		recordedAt = time.Now()
	}

	entry, err := r.db.EntityPriceHistory.Create().
		SetEntityID(entityID).
		SetPrice(price).
		SetMarketLow(low).
		SetMarketMid(mid).
		SetMarketHigh(high).
		SetDirectLow(directLow).
		SetSource(source).
		SetSourceID(sourceID).
		SetRecordedAt(recordedAt).
		SetNotes(notes).
		Save(ctx)
	if err != nil {
		return PriceHistoryEntry{}, err
	}

	_ = r.db.Entity.UpdateOneID(entityID).
		SetCurrentMarketPrice(price).
		SetLastPriceSyncAt(recordedAt).
		Exec(ctx)

	return mapPriceHistoryEntry(entry), nil
}

func (r *PriceHistoryRepository) GetTrackedEntities(ctx context.Context) ([]*ent.Entity, error) {
	return r.db.Entity.Query().
		Where(
			entity.PriceTrackingEnabledEQ(true),
			entity.ArchivedEQ(false),
		).
		WithFields().
		All(ctx)
}

func (r *PriceHistoryRepository) GetTrackedEntitiesByGroup(ctx context.Context, gid uuid.UUID) ([]*ent.Entity, error) {
	return r.db.Entity.Query().
		Where(
			entity.PriceTrackingEnabledEQ(true),
			entity.ArchivedEQ(false),
			entity.HasGroupWith(group.IDEQ(gid)),
		).
		WithFields().
		All(ctx)
}

func (r *PriceHistoryRepository) GetTrackedEntitiesByIDs(ctx context.Context, gid uuid.UUID, ids []uuid.UUID) ([]*ent.Entity, error) {
	return r.db.Entity.Query().
		Where(
			entity.IDIn(ids...),
			entity.ArchivedEQ(false),
			entity.HasGroupWith(group.IDEQ(gid)),
		).
		WithFields().
		All(ctx)
}
