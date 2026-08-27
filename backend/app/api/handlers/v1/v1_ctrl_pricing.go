package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/pricing"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

// HandleItemPricesGet godoc
//
//	@Summary	Get Item Price History
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		id	path	string	true	"Item ID"
//	@Success	200	{array}	repo.PriceHistoryEntry
//	@Router		/v1/entities/{id}/prices [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPricesGet() errchain.HandlerFunc {
	fn := func(r *http.Request, ID uuid.UUID) ([]repo.PriceHistoryEntry, error) {
		auth := services.NewContext(r.Context())
		return ctrl.svc.Pricing.GetPriceHistory(auth, auth.GID, ID)
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleItemPriceSync godoc
//
//	@Summary	Sync Item Price From TCGPlayer/Market
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		id	path		string	true	"Item ID"
//	@Success	200	{object}	repo.PriceHistoryEntry
//	@Router		/v1/entities/{id}/prices/sync [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPriceSync() errchain.HandlerFunc {
	fn := func(r *http.Request, ID uuid.UUID) (repo.PriceHistoryEntry, error) {
		auth := services.NewContext(r.Context())
		entry, err := ctrl.svc.Pricing.SyncEntityPrice(auth, auth.GID, ID)
		if err != nil {
			return repo.PriceHistoryEntry{}, err
		}
		return *entry, nil
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleItemPriceAutoDetect godoc
//
//	@Summary	Auto-detect TCGPlayer Link from Custom Fields and Enable Tracking
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		id	path		string	true	"Item ID"
//	@Success	200	{object}	repo.EntityOut
//	@Router		/v1/entities/{id}/prices/auto-detect [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPriceAutoDetect() errchain.HandlerFunc {
	fn := func(r *http.Request, ID uuid.UUID) (repo.EntityOut, error) {
		auth := services.NewContext(r.Context())
		out, _, err := ctrl.svc.Pricing.AutoDetectEntityTracking(auth, auth.GID, ID)
		if err != nil {
			return repo.EntityOut{}, err
		}
		return *out, nil
	}

	return adapters.CommandID("id", fn, http.StatusOK)
}

// HandleItemPriceCreate godoc
//
//	@Summary	Create Manual Price Snapshot
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		id		path		string					true	"Item ID"
//	@Param		payload	body		repo.PriceHistoryCreate	true	"Price Data"
//	@Success	201		{object}	repo.PriceHistoryEntry
//	@Router		/v1/entities/{id}/prices [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPriceCreate() errchain.HandlerFunc {
	fn := func(r *http.Request, itemID uuid.UUID, body repo.PriceHistoryCreate) (repo.PriceHistoryEntry, error) {
		auth := services.NewContext(r.Context())
		return ctrl.svc.Pricing.CreateManualPrice(auth, auth.GID, itemID, body)
	}

	return adapters.ActionID("id", fn, http.StatusCreated)
}

// HandleItemPriceDelete godoc
//
//	@Summary	Delete Price Snapshot
//	@Tags		Item Pricing
//	@Param		id			path	string	true	"Item ID"
//	@Param		price_id	path	string	true	"Price Snapshot ID"
//	@Success	204
//	@Router		/v1/entities/{id}/prices/{price_id} [DELETE]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPriceDelete() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		itemID, err := adapters.RouteUUID(r, "id")
		if err != nil {
			return err
		}
		priceID, err := adapters.RouteUUID(r, "price_id")
		if err != nil {
			return err
		}
		auth := services.NewContext(r.Context())
		if err := ctrl.svc.Pricing.DeletePrice(auth, auth.GID, itemID, priceID); err != nil {
			return err
		}
		return server.JSON(w, http.StatusNoContent, nil)
	}
}

type ProductPricingSearchQuery struct {
	Query string `schema:"q"`
}

// HandleProductPricingSearch godoc
//
//	@Summary	Search Products in TCG Catalog
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		q	query	string	true	"Search query"
//	@Success	200	{array}	pricing.ProductSearchResult
//	@Router		/v1/products/search-pricing [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleProductPricingSearch() errchain.HandlerFunc {
	fn := func(r *http.Request, q ProductPricingSearchQuery) ([]pricing.ProductSearchResult, error) {
		auth := services.NewContext(r.Context())
		return ctrl.svc.Pricing.SearchProducts(auth, q.Query)
	}

	return adapters.Query(fn, http.StatusOK)
}

type SyncPricesResponse struct {
	UpdatedCount int `json:"updatedCount"`
}

type SyncPricesBulkRequest struct {
	EntityIDs []uuid.UUID `json:"entityIds"`
}

// HandleItemPricesSyncAll godoc
//
//	@Summary	Sync All Tracked Items Market Prices for Group
//	@Tags		Item Pricing
//	@Produce	json
//	@Success	200	{object}	SyncPricesResponse
//	@Router		/v1/entities/prices/sync-all [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPricesSyncAll() errchain.HandlerFunc {
	fn := func(r *http.Request) (SyncPricesResponse, error) {
		auth := services.NewContext(r.Context())
		count, err := ctrl.svc.Pricing.SyncGroupTrackedEntities(auth, auth.GID)
		if err != nil {
			return SyncPricesResponse{}, err
		}
		return SyncPricesResponse{UpdatedCount: count}, nil
	}

	return adapters.Command(fn, http.StatusOK)
}

// HandleItemPricesSyncBulk godoc
//
//	@Summary	Sync Selected Items Market Prices in Bulk
//	@Tags		Item Pricing
//	@Produce	json
//	@Param		payload	body		SyncPricesBulkRequest	true	"Entity IDs to sync"
//	@Success	200		{object}	SyncPricesResponse
//	@Router		/v1/entities/prices/sync-bulk [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleItemPricesSyncBulk() errchain.HandlerFunc {
	fn := func(r *http.Request, body SyncPricesBulkRequest) (SyncPricesResponse, error) {
		auth := services.NewContext(r.Context())
		count, err := ctrl.svc.Pricing.SyncEntitiesBulk(auth, auth.GID, body.EntityIDs)
		if err != nil {
			return SyncPricesResponse{}, err
		}
		return SyncPricesResponse{UpdatedCount: count}, nil
	}

	return adapters.Action(fn, http.StatusOK)
}
