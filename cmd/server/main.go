package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmitriy-dev/hotel-content-aggregator/internal/handler"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/model"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/repository"
	"github.com/dmitriy-dev/hotel-content-aggregator/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🏨 Starting Hotel Content Aggregator Service...")

	// Storage layer (in-memory engine with PostgreSQL interface compatibility)
	repo := repository.NewMemoryHotelRepository()
	hotelService := service.NewHotelService(repo)

	// Seed sample partner data for instant testing
	seedInitialCatalog(hotelService)

	// Routing & Middleware
	mux := http.NewServeMux()
	h := handler.NewHandler(hotelService)
	h.RegisterRoutes(mux)

	var appHandler http.Handler = mux
	appHandler = handler.LoggingMiddleware(appHandler)
	appHandler = handler.RecoveryMiddleware(appHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      appHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server startup in background goroutine
	go func() {
		log.Printf("🌍 Server listening on http://localhost:%s", port)
		log.Printf("👉 Healthcheck: http://localhost:%s/health", port)
		log.Printf("👉 Search API:  http://localhost:%s/api/v1/hotels?city=Москва", port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal server error: %v", err)
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("\n⚠️ Termination signal received. Initiating Graceful Shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown with error: %v", err)
	}

	log.Println("✅ Server gracefully exited. All connections drained.")
}

func seedInitialCatalog(svc *service.HotelService) {
	ctx := context.Background()

	samples := []model.IngestPayload{
		{
			SupplierID:  "ostrovok",
			ExternalID:  "ost-001",
			Name:        "Radisson Blu Olympiyskiy",
			City:        "Москва",
			Address:     "Самарская ул., 1",
			Latitude:    55.7876,
			Longitude:   37.6258,
			Stars:       5,
			Rating:      9.1,
			Description: "Премиальный отель с видом на центр столицы и спортивный комплекс Олимпийский.",
			Amenities:   []string{"WiFi", "Спа", "Фитнес-центр", "Ресторан", "Подземный паркинг"},
			Photos:      []string{"https://images.hotel.example/radisson1.jpg"},
		},
		{
			SupplierID:  "bronevik",
			ExternalID:  "brn-042",
			Name:        "Гранд Отель Европа",
			City:        "Санкт-Петербург",
			Address:     "Михайловская ул., 1/7",
			Latitude:    59.9355,
			Longitude:   30.3312,
			Stars:       5,
			Rating:      9.4,
			Description: "Исторический отель в самом сердце Санкт-Петербурга, рядом с Невским проспектом.",
			Amenities:   []string{"WiFi", "Консьерж-сервис", "Исторические интерьеры", "Завтрак 'Шведский стол'"},
			Photos:      []string{"https://images.hotel.example/europe1.jpg"},
		},
	}

	for _, s := range samples {
		_, _ = svc.Ingest(ctx, s)
	}
	log.Println("📦 Seeded initial test dataset (2 verified hotels from Ostrovok & Bronevik)")
}
