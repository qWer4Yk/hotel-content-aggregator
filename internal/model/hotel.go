package model

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidHotelData   = errors.New("hotel: invalid hotel data")
	ErrHotelNotFound      = errors.New("hotel: not found")
	ErrDuplicateSupplier  = errors.New("hotel: duplicate supplier entry")
)

// Hotel represents a normalized hotel entity in the catalog.
type Hotel struct {
	ID             string    `json:"id"`
	SupplierID     string    `json:"supplier_id"`
	ExternalID     string    `json:"external_id"`
	Name           string    `json:"name"`
	NormalizedName string    `json:"normalized_name"`
	City           string    `json:"city"`
	Address        string    `json:"address"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Stars          int       `json:"stars"`
	Rating         float64   `json:"rating"`
	Description    string    `json:"description"`
	Amenities      []string  `json:"amenities"`
	Photos         []string  `json:"photos"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IngestPayload represents raw hotel data received from external partners.
type IngestPayload struct {
	SupplierID  string   `json:"supplier_id"`
	ExternalID  string   `json:"external_id"`
	Name        string   `json:"name"`
	City        string   `json:"city"`
	Address     string   `json:"address"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Stars       int      `json:"stars"`
	Rating      float64  `json:"rating"`
	Description string   `json:"description"`
	Amenities   []string `json:"amenities"`
	Photos      []string `json:"photos"`
}

// Validate checks basic sanity of ingested payload.
func (p *IngestPayload) Validate() error {
	if strings.TrimSpace(p.SupplierID) == "" {
		return errors.New("supplier_id is required")
	}
	if strings.TrimSpace(p.ExternalID) == "" {
		return errors.New("external_id is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("hotel name is required")
	}
	if strings.TrimSpace(p.City) == "" {
		return errors.New("city is required")
	}
	if p.Stars < 0 || p.Stars > 5 {
		return errors.New("stars must be between 0 and 5")
	}
	if p.Rating < 0 || p.Rating > 10 {
		return errors.New("rating must be between 0 and 10")
	}
	return nil
}

// SearchFilter specifies criteria for catalog querying.
type SearchFilter struct {
	City      string  `json:"city"`
	MinStars  int     `json:"min_stars"`
	MinRating float64 `json:"min_rating"`
	Limit     int     `json:"limit"`
	Offset    int     `json:"offset"`
}

// NormalizeName prepares hotel name for deduplication (lowercasing, trimming, removing quotes).
func NormalizeName(name string) string {
	cleaned := strings.ToLower(strings.TrimSpace(name))
	cleaned = strings.ReplaceAll(cleaned, "\"", "")
	cleaned = strings.ReplaceAll(cleaned, "«", "")
	cleaned = strings.ReplaceAll(cleaned, "»", "")
	return cleaned
}
