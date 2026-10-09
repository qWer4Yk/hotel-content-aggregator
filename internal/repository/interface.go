package repository

import (
	"context"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
)

// HotelRepository defines data access contracts for hotel catalog operations.
type HotelRepository interface {
	Save(ctx context.Context, hotel *model.Hotel) error
	GetByID(ctx context.Context, id string) (*model.Hotel, error)
	FindBySupplierKey(ctx context.Context, supplierID, externalID string) (*model.Hotel, error)
	FindMatch(ctx context.Context, normalizedName, city string, lat, lon float64) (*model.Hotel, error)
	Search(ctx context.Context, filter model.SearchFilter) ([]model.Hotel, int, error)
	TotalCount(ctx context.Context) (int, error)
}
