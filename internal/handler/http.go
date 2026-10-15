package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/service"
)

// Handler routes HTTP requests and coordinates presentation logic.
type Handler struct {
	svc *service.HotelService
}

// NewHandler initializes router handler.
func NewHandler(svc *service.HotelService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes configures endpoints using Go standard HTTP router.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /api/v1/hotels/ingest", h.IngestHotel)
	mux.HandleFunc("POST /api/v1/hotels/batch-ingest", h.BatchIngest)
	mux.HandleFunc("GET /api/v1/hotels", h.SearchHotels)
	mux.HandleFunc("GET /api/v1/hotels/{id}", h.GetHotelByID)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) IngestHotel(w http.ResponseWriter, r *http.Request) {
	var payload model.IngestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	result, err := h.svc.Ingest(r.Context(), payload)
	if err != nil {
		if errors.Is(err, model.ErrInvalidHotelData) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to ingest hotel: "+err.Error())
		return
	}

	status := http.StatusCreated
	if result.IsUpdated {
		status = http.StatusOK
	}

	writeJSON(w, status, result)
}

func (h *Handler) BatchIngest(w http.ResponseWriter, r *http.Request) {
	var payloads []model.IngestPayload
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON array payload: "+err.Error())
		return
	}

	if len(payloads) > 500 {
		writeError(w, http.StatusRequestEntityTooLarge, "Batch limit is 500 items per request")
		return
	}

	type summary struct {
		Processed int `json:"processed"`
		Created   int `json:"created"`
		Updated   int `json:"updated"`
		Errors    int `json:"errors"`
	}

	var stats summary
	stats.Processed = len(payloads)

	for _, p := range payloads {
		res, err := h.svc.Ingest(r.Context(), p)
		if err != nil {
			stats.Errors++
			continue
		}
		if res.IsUpdated {
			stats.Updated++
		} else {
			stats.Created++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Batch ingestion finished",
		"stats":   stats,
	})
}

func (h *Handler) SearchHotels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.SearchFilter{
		City:   q.Get("city"),
		Limit:  10,
		Offset: 0,
	}

	if val := q.Get("min_stars"); val != "" {
		if s, err := strconv.Atoi(val); err == nil {
			filter.MinStars = s
		}
	}
	if val := q.Get("min_rating"); val != "" {
		if rate, err := strconv.ParseFloat(val, 64); err == nil {
			filter.MinRating = rate
		}
	}
	if val := q.Get("limit"); val != "" {
		if l, err := strconv.Atoi(val); err == nil {
			filter.Limit = l
		}
	}
	if val := q.Get("offset"); val != "" {
		if o, err := strconv.Atoi(val); err == nil {
			filter.Offset = o
		}
	}

	hotels, total, err := h.svc.Search(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   hotels,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) GetHotelByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "Hotel ID is missing in path")
		return
	}

	hotel, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrHotelNotFound) {
			writeError(w, http.StatusNotFound, "Hotel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Error retrieving hotel: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, hotel)
}

// LoggingMiddleware logs method, path, and duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s - %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

// RecoveryMiddleware protects against server crashes from panics.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("🔥 CRITICAL PANIC RECOVERED: %v", rec)
				writeError(w, http.StatusInternalServerError, "Internal Server Panic Recovered")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":  message,
		"code":   status,
		"status": http.StatusText(status),
	})
}
