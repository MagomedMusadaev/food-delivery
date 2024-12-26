package handler

import (
	"encoding/json"
	"fmt"
	"food-delivery/internal/auth/entities"
	"food-delivery/internal/auth/service"
	"food-delivery/pkg/logger"
	"food-delivery/pkg/utils"
	"net/http"
	"time"
)

type AuthHandlerInt interface {
	Register(w http.ResponseWriter, r *http.Request)
	ConfirmEmail(w http.ResponseWriter, r *http.Request)
	SignIn(w http.ResponseWriter, r *http.Request)
	RefreshTokens(w http.ResponseWriter, r *http.Request)
	SignOut(w http.ResponseWriter, r *http.Request)
}

type AuthHandler struct {
	service service.AuthServiceInt
	log     *logger.Logger
}

func NewAuthHandler(service service.AuthServiceInt, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		log:     log,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user entities.User

	// Декодируем тело запроса в структуру User.
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.log.Error("Ошибка декодирования JSON: ", err)
		utils.DecodeErr(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Вызов сервис слоя для 1 этапа регистрации пользователя.
	if err := h.service.Register(&user); err != nil {
		utils.DecodeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Формируем и отправляем успешный ответ.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	message := fmt.Sprintf("Письмо с кодом подтверждения отправлено на почту %s", user.Email)
	if err := json.NewEncoder(w).Encode(entities.Response{Message: message}); err != nil {
		h.log.Error("Ошибка при отправке ответа: ", err)
	}
}

func (h *AuthHandler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	// Декодируем тело запроса в структуру ConfirmEmailRequest.
	var request entities.ConfirmEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.log.Error("Ошибка декодирования JSON в ConfirmEmail: ", err)
		utils.DecodeErr(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	// Проверяем код подтверждения через слой сервиса.
	if err := h.service.ConfirmEmail(request.Code); err != nil {
		utils.DecodeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Устанавливаем заголовки ответа.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Создаём ответное сообщение.
	response := entities.Response{
		Message: "Электронная почта подтверждена",
	}

	// Отправляем JSON-ответ.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Ошибка при отправке ответа: ", err)
	}
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	// Декодируем тело запроса в структуру User.
	var user entities.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.log.Error("Ошибка декодирования JSON в User: ", err)
		utils.DecodeErr(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	userAddr := r.RemoteAddr

	// Вызов сервис слоя для авторизации пользователя.
	tokens, err := h.service.SignIn(&user, userAddr)
	if err != nil {
		utils.DecodeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Устанавливаем refresh токен в cookies.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // Время жизни куки (30 дней)
		HttpOnly: false,                               // Защита от доступа через JavaScript
		Secure:   false,                               // Для использования в HTTP (для HTTPS изменить на true)
		Path:     "/",                                 // Путь для куки
	})

	// Устанавливаем заголовки ответа.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Отправляем данные с access токеном в JSON-формате.
	err = json.NewEncoder(w).Encode(&tokens)
	if err != nil {
		h.log.Error("Ошибка при отправке ответа: ", err)
		utils.DecodeErr(w, "Ошибка отправки данных", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	// Получаем refresh токен из cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		h.log.Error("не удалось получить данные из cookie:", err)
		utils.DecodeErr(w, "ошибка авторизации", http.StatusBadRequest)
		return
	}

	// Вызов сервис слоя для обновления токенов
	tokens, err := h.service.RefreshTokens(cookie.Value)
	if err != nil {
		utils.DecodeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Устанавливаем refresh токен в cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // Время жизни куки (30 дней)
		HttpOnly: false,                               // Защита от доступа через JavaScript
		Secure:   false,                               // Для использования в HTTP (для HTTPS изменить на true)
		Path:     "/",                                 // Путь для куки
	})

	// Устанавливаем заголовки ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Отправляем данные с access токеном в JSON-формате
	err = json.NewEncoder(w).Encode(&tokens)
	if err != nil {
		h.log.Error("Ошибка при отправке ответа: ", err)
		utils.DecodeErr(w, "Ошибка отправки данных", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) SignOut(w http.ResponseWriter, r *http.Request) {
	// Получаем refresh токен из cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		h.log.Error("не удалось получить данные из cookie:", err)
		utils.DecodeErr(w, "ошибка авторизации", http.StatusBadRequest)
		return
	}

	if err := h.service.SignOut(cookie.Value); err != nil {
		utils.DecodeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Удаляем refresh токен в cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Unix(0, 0), // Устанавливаем время истечения в прошлое, чтобы удалить куку
		HttpOnly: false,           // Защита от доступа через JavaScript
		Secure:   false,           // Для использования в HTTP (для HTTPS изменить на true)
		Path:     "/",             // Путь для куки
	})

	// Устанавливаем заголовки ответа.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Отправляем ответ с сообщением.
	message := fmt.Sprintf("Выход выполнен успешно")
	if err := json.NewEncoder(w).Encode(entities.Response{Message: message}); err != nil {
		h.log.Error("Ошибка при отправке ответа: ", err)
	}
}
