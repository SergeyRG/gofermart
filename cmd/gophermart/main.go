package main

import (
	"log"
	"net/http"
	"time"

	"github.com/SergeyRG/gofermart/internal/config"
	"github.com/SergeyRG/gofermart/internal/handlers"
	"github.com/SergeyRG/gofermart/internal/logging"
	"github.com/SergeyRG/gofermart/internal/migrations"
	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/repositories"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var USERID model.UserID

func main() {
	USERID = 1 //TODO

	// Инициализация логгера
	err := logging.Initialize("DEBUG")
	if err != nil {
		log.Fatalf("ошибка инициализации системы логгирования: %v", err)
	}
	l := logging.Logger

	l.Info("чтение конфигурационной информации")
	cfg, err := config.NewConfig()
	if err != nil {
		l.Fatal(
			"Сбой запуска приложения, не удалось ошибка конфигурации", zap.Error(err))
	}
	l.Info("конфигурационная информация прочитана")

	l.Info("Обновление БД")
	err = migrations.RunMigrations(cfg.DBDSN)
	if err != nil {
		l.Fatal(
			"Сбой запуска приложения, не удалось миграцию схемы БД", zap.Error(err))
	}
	l.Info("Обновление БД завершено")

	//Создание сервиса обработки заказов
	orderRepo, err := repositories.NewPSQLOrderRepo(cfg.DBDSN)
	if err != nil {
		l.Fatal(
			"Ошибка подключения к БД", zap.Error(err))
	}
	orderSvc := services.NewOrderService(orderRepo)

	if err := run(cfg, orderSvc); err != nil {
		log.Fatalf("ошибка запуска приложения: %v", err)
	}
}

func run(cfg config.Config, orderSvc services.OrderService) error {
	l := logging.Logger
	l.Info("Инициализация http сервера")

	r := chi.NewRouter()

	OrderHandler := handlers.NewOrderHandler(orderSvc)

	r.Route("/", func(r chi.Router) {
		// 	r.Use(logging.WithLogging)
		// 	r.Use(authMiddleware)
		// 	r.Use(middleware.GzipMiddleware)
		// 	r.Post("/", rootHandler)
		// 	r.Get("/{id}", redirectHandler)
		// 	r.Get("/{id}/", redirectHandler)
		// 	r.Get("/ping", DBPingHandler)
		// 	r.Get("/ping/", DBPingHandler)
		// 	r.Post("/api/shorten", JSONShortenHandler)
		r.Post("/api/user/orders", OrderHandler.AddNewOrder())
		// 	r.Get("/api/user/urls", UserURLHandler)
		// 	r.Delete("/api/user/urls", UserBatchDeleteHandler)
	})

	server := &http.Server{
		Addr:              cfg.ServerAddress,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	l.Info("Запуск http сервера")
	return server.ListenAndServe()
}

// func initRouter() chi.Router {

// 	// redirectHandler := handler.RedirectHandler(svc)
// 	// JSONShortenHandler := handler.JSONShortenHandler(svc)
// 	// DBPingHandler := handler.DBPingHandler(db)
// 	// BatchAddHandler := handler.BatchAddHandler(svc)
// 	// UserURLHandler := handler.UserURLHandler(svc)
// 	// UserBatchDeleteHandler := handler.UserBatchDeleteHandler(svc)

// 	// authMiddleware := middleware.Auth(cfg)

// 	// r.Route("/", func(r chi.Router) {
// 	// 	r.Use(logging.WithLogging)
// 	// 	r.Use(authMiddleware)
// 	// 	r.Use(middleware.GzipMiddleware)
// 	// 	r.Post("/", rootHandler)
// 	// 	r.Get("/{id}", redirectHandler)
// 	// 	r.Get("/{id}/", redirectHandler)
// 	// 	r.Get("/ping", DBPingHandler)
// 	// 	r.Get("/ping/", DBPingHandler)
// 	// 	r.Post("/api/shorten", JSONShortenHandler)
// 	// 	r.Post("/api/shorten/batch", BatchAddHandler)
// 	// 	r.Get("/api/user/urls", UserURLHandler)
// 	// 	r.Delete("/api/user/urls", UserBatchDeleteHandler)
// 	// })
// 	return chi.NewRouter()
// }
