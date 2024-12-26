package main

import (
	"database/sql"
	"food-delivery/internal/auth/handler"
	"food-delivery/internal/auth/repository"
	"food-delivery/internal/auth/service"
	"food-delivery/pkg/logger"
	"github.com/redis/go-redis/v9"
)

func initAuthModule(db *sql.DB, client *redis.Client, log *logger.Logger) *handler.AuthHandler {
	authRepository := repository.NewAuthRepository(db, log)
	authService := service.NewAuthService(authRepository, client, log)
	authHandler := handler.NewAuthHandler(authService, log)
	return authHandler
}

//func initRestaurantModule(db, client, log) {
//
//}
