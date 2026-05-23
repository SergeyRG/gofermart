package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/SergeyRG/gofermart/internal/auth"
	"github.com/SergeyRG/gofermart/internal/clients"
	"github.com/SergeyRG/gofermart/internal/config"
	"github.com/SergeyRG/gofermart/internal/handlers"
	"github.com/SergeyRG/gofermart/internal/logging"
	"github.com/SergeyRG/gofermart/internal/middleware"
	"github.com/SergeyRG/gofermart/internal/migrations"
	"github.com/SergeyRG/gofermart/internal/repositories"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {

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
			"Сбой запуска приложения, ошибка конфигурации", zap.Error(err))
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
	db, err := repositories.CreateAndCheckPSQLCon(cfg.DBDSN)
	if err != nil {
		l.Fatal(
			"Ошибка подключения к БД", zap.Error(err))
	}
	txManager := repositories.NewTXManager(db)

	orderRepo := repositories.NewPSQLOrderRepo(db)
	orderSvc := services.NewOrderService(orderRepo, txManager, 6)

	balanceRepo := repositories.NewPSQLBalanceRepo(db)
	balanceSvc := services.NewBalanceService(balanceRepo, txManager)

	accrualHTTPClient := clients.NewRestyAccrualClient(cfg)
	accrualSvc := services.NewAccrualService(txManager, orderSvc, balanceSvc, accrualHTTPClient)

	userRepo := repositories.NewPSQLUserRepo(db)
	userSvc := services.NewUserService(userRepo, txManager, balanceSvc)

	g, gCtx := errgroup.WithContext(context.Background())

	for workerID := range orderSvc.WorkersCount {
		g.Go(func() error {
			return accrualSvc.RunWorker(gCtx, workerID)
		})
	}

	g.Go(func() error {
		if err := run(cfg, orderSvc, balanceSvc, userSvc); err != nil {
			l.Error("ошибка запуска приложения", zap.Error(err))
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		l.Fatal("критическая ошибка, повлекшая остановку приложения.", zap.Error(err))
	}
}

func run(cfg config.Config,
	orderSvc services.OrderService,
	balanceSvc services.BalanceService,
	userSvc services.UserService) error {

	l := logging.Logger
	l.Info("Инициализация http сервера")

	r := chi.NewRouter()

	jwtm := auth.NewJWTManager([]byte(cfg.SecretKey))
	UserHandlers := handlers.NewUserHandler(userSvc, *jwtm)
	r.Post("/api/user/register", UserHandlers.Register())
	r.Post("/api/user/login", UserHandlers.Login())

	StandartHandlers := handlers.NewStandartHandlers(orderSvc, balanceSvc)
	authMiddleware := middleware.Auth(*jwtm)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/api/user/balance", StandartHandlers.GetUserBalance())
		r.Post("/api/user/orders", StandartHandlers.AddNewOrder())
		r.Get("/api/user/orders", StandartHandlers.GetUserOrders())
		r.Post("/api/user/balance/withdraw", StandartHandlers.Withdraw())
		r.Get("/api/user/withdrawals", StandartHandlers.GetWithdrawals())
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
