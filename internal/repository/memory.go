package repository

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
)

// MemoryHotelRepository is an in-memory, thread-safe implementation of HotelRepository.
type MemoryHotelRepository struct {
	mu     sync.RWMutex
	hotels map[string]*model.Hotel
}

// NewMemoryHotelRepository initializes a storage backed by memory.
func NewMemoryHotelRepository() *MemoryHotelRepository {
	return &MemoryHotelRepository{
		hotels: make(map[string]*model.Hotel),
	}
}

func (r *MemoryHotelRepository) Save(ctx context.Context, h *model.Hotel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if h.CreatedAt.IsZero() {
		h.CreatedAt = now
	}
	h.UpdatedAt = now

	// Clone to prevent external mutations
	stored := *h
	r.hotels[h.ID] = &stored
	return nil
}

func (r *MemoryHotelRepository) GetByID(ctx context.Context, id string) (*model.Hotel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, exists := r.hotels[id]
	if !exists {
		return nil, model.ErrHotelNotFound
	}
	copyH := *h
	return &copyH, nil
}

func (r *MemoryHotelRepository) FindBySupplierKey(ctx context.Context, supplierID, externalID string) (*model.Hotel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.hotels {
		if h.SupplierID == supplierID && h.ExternalID == externalID {
			copyH := *h
			return &copyH, nil
		}
	}
	return nil, model.ErrHotelNotFound
}

// FindMatch looks for existing hotel with matching name in the same city or within ~200m distance.
func (r *MemoryHotelRepository) FindMatch(ctx context.Context, normalizedName, city string, lat, lon float64) (*model.Hotel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	targetCity := strings.ToLower(strings.TrimSpace(city))

	for _, h := range r.hotels {
		sameCity := strings.ToLower(h.City) == targetCity
		sameName := h.NormalizedName == normalizedName

		if sameCity && sameName {
			copyH := *h
			return &copyH, nil
		}

		// Distance check if coordinates are present (< 200 meters)
		if lat != 0 && lon != 0 && h.Latitude != 0 && h.Longitude != 0 {
			dist := haversineDistanceMeters(lat, lon, h.Latitude, h.Longitude)
			if dist <= 200 && sameName {
				copyH := *h
				return &copyH, nil
			}
		}
	}

	return nil, model.ErrHotelNotFound
}

func (r *MemoryHotelRepository) Search(ctx context.Context, filter model.SearchFilter) ([]model.Hotel, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*model.Hotel
	cityFilter := strings.ToLower(strings.TrimSpace(filter.City))

	for _, h := range r.hotels {
		if cityFilter != "" && !strings.Contains(strings.ToLower(h.City), cityFilter) {
			continue
		}
		if filter.MinStars > 0 && h.Stars < filter.MinStars {
			continue
		}
		if filter.MinRating > 0 && h.Rating < filter.MinRating {
			continue
		}
		matched = append(matched, h)
	}

	// Sort by rating desc
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Rating > matched[j].Rating
	})

	total := len(matched)
	if filter.Offset >= total {
		return []model.Hotel{}, total, nil
	}

	end := filter.Offset + filter.Limit
	if filter.Limit <= 0 || end > total {
		end = total
	}

	result := make([]model.Hotel, 0, end-filter.Offset)
	for i := filter.Offset; i < end; i++ {
		result = append(result, *matched[i])
	}

	return result, total, nil
}

func (r *MemoryHotelRepository) TotalCount(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.hotels), nil
}

// haversineDistanceMeters calculates approximate spherical distance between coordinates.
func haversineDistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
