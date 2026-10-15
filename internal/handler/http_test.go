package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/repository"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/service"
)

func setupTestRouter() http.Handler {
	repo := repository.NewMemoryHotelRepository()
	svc := service.NewHotelService(repo)
	h := NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestHandler_Health(t *testing.T) {
	router := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if body["status"] != "healthy" {
		t.Errorf("expected status healthy, got %s", body["status"])
	}
}

func TestHandler_IngestAndSearch(t *testing.T) {
	router := setupTestRouter()

	// Ingest hotel via POST
	payload := model.IngestPayload{
		SupplierID: "tutu-partner",
		ExternalID: "ext-1",
		Name:       "Тестовый Спа Отель",
		City:       "Сочи",
		Stars:      4,
		Rating:     9.0,
	}

	raw, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/hotels/ingest", bytes.NewReader(raw))
	postRec := httptest.NewRecorder()

	router.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", postRec.Code, postRec.Body.String())
	}

	// Search hotel via GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/hotels?city=Сочи", nil)
	getRec := httptest.NewRecorder()

	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getRec.Code)
	}

	var searchResp struct {
		Data  []model.Hotel `json:"data"`
		Total int           `json:"total"`
	}
	if err := json.NewDecoder(getRec.Body).Decode(&searchResp); err != nil {
		t.Fatalf("failed to decode search response: %v", err)
	}

	if searchResp.Total != 1 {
		t.Errorf("expected 1 hotel found, got %d", searchResp.Total)
	}
	if searchResp.Data[0].Name != "Тестовый Спа Отель" {
		t.Errorf("unexpected hotel name: %s", searchResp.Data[0].Name)
	}
}
