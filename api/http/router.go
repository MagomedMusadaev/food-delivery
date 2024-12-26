package http

import (
	"food-delivery/internal/auth/handler"
	"github.com/gorilla/mux"
)

func InitRoutes(r *mux.Router, authHandler *handler.AuthHandler) {
	// Эндпоинты модуля auth
	r.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/auth/confirm-email", authHandler.ConfirmEmail).Methods("GET")
	r.HandleFunc("/auth/sign-in", authHandler.SignIn).Methods("POST")
	r.HandleFunc("/auth/sign-out", authHandler.SignOut).Methods("POST")
	r.HandleFunc("/auth/refresh", authHandler.RefreshTokens).Methods("POST")

	// Эндпоинты модуля restaurant
	//r.HandleFunc()...

}
