package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/repository"
)

// HotelService orchestrates business rules, normalization and deduplication.
type HotelService struct {
	repo repository.HotelRepository
}

// NewHotelService constructs a new service instance.
func NewHotelService(repo repository.HotelRepository) *HotelService {
	return &HotelService{repo: repo}
}

// IngestResult details the outcome of an ingest operation.
type IngestResult struct {
	Hotel     *model.Hotel `json:"hotel"`
	IsUpdated bool         `json:"is_updated"`
	MatchedID string       `json:"matched_id,omitempty"`
}

// Ingest processes a single partner payload with validation and deduplication.
func (s *HotelService) Ingest(ctx context.Context, p model.IngestPayload) (*IngestResult, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", model.ErrInvalidHotelData, err.Error())
	}

	normName := model.NormalizeName(p.Name)

	// Step 1: Check if exact supplier key already exists (re-ingestion/update)
	existing, err := s.repo.FindBySupplierKey(ctx, p.SupplierID, p.ExternalID)
	if err == nil && existing != nil {
		// Update fields with latest content
		existing.Name = p.Name
		existing.NormalizedName = normName
		existing.Address = p.Address
		existing.Latitude = p.Latitude
		existing.Longitude = p.Longitude
		existing.Stars = p.Stars
		existing.Rating = p.Rating
		existing.Description = p.Description
		existing.Amenities = mergeSlices(existing.Amenities, p.Amenities)
		existing.Photos = mergeSlices(existing.Photos, p.Photos)

		if err := s.repo.Save(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update hotel: %w", err)
		}
		return &IngestResult{Hotel: existing, IsUpdated: true}, nil
	}

	// Step 2: Check deduplication across other suppliers in the same city
	match, err := s.repo.FindMatch(ctx, normName, p.City, p.Latitude, p.Longitude)
	if err == nil && match != nil {
		// Found existing hotel from another supplier - enrich existing content
		if len(p.Description) > len(match.Description) {
			match.Description = p.Description
		}
		match.Amenities = mergeSlices(match.Amenities, p.Amenities)
		match.Photos = mergeSlices(match.Photos, p.Photos)
		if p.Rating > match.Rating {
			match.Rating = p.Rating
		}

		if err := s.repo.Save(ctx, match); err != nil {
			return nil, fmt.Errorf("failed to enrich matched hotel: %w", err)
		}
		return &IngestResult{
			Hotel:     match,
			IsUpdated: true,
			MatchedID: match.ID,
		}, nil
	}

	// Step 3: Fresh new hotel
	newID := generateID()
	newHotel := &model.Hotel{
		ID:             newID,
		SupplierID:     p.SupplierID,
		ExternalID:     p.ExternalID,
		Name:           p.Name,
		NormalizedName: normName,
		City:           p.City,
		Address:        p.Address,
		Latitude:       p.Latitude,
		Longitude:      p.Longitude,
		Stars:          p.Stars,
		Rating:         p.Rating,
		Description:    p.Description,
		Amenities:      p.Amenities,
		Photos:         p.Photos,
	}

	if err := s.repo.Save(ctx, newHotel); err != nil {
		return nil, fmt.Errorf("failed to save new hotel: %w", err)
	}

	return &IngestResult{Hotel: newHotel, IsUpdated: false}, nil
}

// Search queries catalog with pagination and filters.
func (s *HotelService) Search(ctx context.Context, filter model.SearchFilter) ([]model.Hotel, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // Protection against huge payload scans
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.Search(ctx, filter)
}

// GetByID retrieves a single hotel.
func (s *HotelService) GetByID(ctx context.Context, id string) (*model.Hotel, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func mergeSlices(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))

	for _, item := range a {
		item = strings.TrimSpace(item)
		if item != "" {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				result = append(result, item)
			}
		}
	}
	for _, item := range b {
		item = strings.TrimSpace(item)
		if item != "" {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				result = append(result, item)
			}
		}
	}
	return result
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
