package service

import (
	"context"
	"testing"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/repository"
)

func TestHotelService_IngestAndDedup(t *testing.T) {
	repo := repository.NewMemoryHotelRepository()
	svc := NewHotelService(repo)
	ctx := context.Background()

	// 1. Supplier A ingests "Cosmos Hotel Moscow"
	payloadA := model.IngestPayload{
		SupplierID:  "ostrovok",
		ExternalID:  "ost-101",
		Name:        "«Космос» Отель",
		City:        "Москва",
		Address:     "Проспект Мира, 150",
		Latitude:    55.8225,
		Longitude:   37.6472,
		Stars:       4,
		Rating:      8.2,
		Description: "Легендарная гостиница у ВДНХ",
		Amenities:   []string{"WiFi", "Парковка"},
		Photos:      []string{"https://img.example/cosmos1.jpg"},
	}

	resA, err := svc.Ingest(ctx, payloadA)
	if err != nil {
		t.Fatalf("unexpected ingest error for supplier A: %v", err)
	}
	if resA.IsUpdated {
		t.Errorf("expected new hotel, got updated")
	}
	firstID := resA.Hotel.ID

	// 2. Supplier B ingests same hotel with slightly different name formatting
	payloadB := model.IngestPayload{
		SupplierID:  "bronevik",
		ExternalID:  "brn-8899",
		Name:        "космос отель",
		City:        "Москва",
		Address:     "пр-т Мира, 150",
		Latitude:    55.8226,
		Longitude:   37.6473,
		Stars:       4,
		Rating:      8.6, // Higher rating
		Description: "Легендарная гостиница у ВДНХ с панорамным видом на Москву",
		Amenities:   []string{"WiFi", "Бассейн", "Завтрак"},
		Photos:      []string{"https://img.example/cosmos2.jpg"},
	}

	resB, err := svc.Ingest(ctx, payloadB)
	if err != nil {
		t.Fatalf("unexpected ingest error for supplier B: %v", err)
	}

	if !resB.IsUpdated {
		t.Errorf("expected deduplication match and update, got new hotel")
	}
	if resB.Hotel.ID != firstID {
		t.Errorf("expected matched ID %s, got %s", firstID, resB.Hotel.ID)
	}

	// Verify enriched attributes
	if resB.Hotel.Rating != 8.6 {
		t.Errorf("expected higher rating 8.6, got %f", resB.Hotel.Rating)
	}
	if len(resB.Hotel.Amenities) != 4 { // WiFi, Парковка, Бассейн, Завтрак
		t.Errorf("expected 4 merged amenities, got %d: %v", len(resB.Hotel.Amenities), resB.Hotel.Amenities)
	}

	// 3. Search validation
	hotels, total, err := svc.Search(ctx, model.SearchFilter{
		City:      "Москва",
		MinRating: 8.0,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 unique hotel in catalog, got %d", total)
	}
	if hotels[0].ID != firstID {
		t.Errorf("search returned unexpected hotel ID")
	}
}

func TestHotelService_Validation(t *testing.T) {
	repo := repository.NewMemoryHotelRepository()
	svc := NewHotelService(repo)
	ctx := context.Background()

	badPayload := model.IngestPayload{
		SupplierID: "", // Missing
		Name:       "Some Hotel",
	}

	_, err := svc.Ingest(ctx, badPayload)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
