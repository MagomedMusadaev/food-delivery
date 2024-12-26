package service

import (
	"context"
	"encoding/json"
	"errors"
	"food-delivery/internal/auth/entities"
	"food-delivery/internal/auth/repository"
	"food-delivery/pkg/logger"
	"food-delivery/pkg/middlewares"
	"food-delivery/pkg/utils"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"net"
	"time"
)

var (
	ttl                     = time.Minute * 2
	ctx                     = context.Background()
	errInternal             = errors.New("произошла внутренняя ошибка")
	errInvalidData          = errors.New("невалидные данные")
	errIncorrectPasAndEmail = errors.New("неверный email или пароль")
	ttlAccess               = time.Minute * 15      // 15 минут
	ttlRefresh              = (time.Hour * 24) * 30 // 30 дней
)

type AuthServiceInt interface {
	Register(user *entities.User) error
	ConfirmEmail(code string) error
	SignIn(user *entities.User, userAddr string) (*entities.TokensResponse, error)
	RefreshTokens(refreshToken string) (*entities.TokensResponse, error)
	SignOut(refreshToken string) error
}

type AuthService struct {
	repo   repository.AuthRepoInt
	client *redis.Client
	log    *logger.Logger
}

func NewAuthService(repo repository.AuthRepoInt, client *redis.Client, log *logger.Logger) *AuthService {
	return &AuthService{
		repo:   repo,
		client: client,
		log:    log,
	}
}

func (s *AuthService) Register(user *entities.User) error {
	// Валидируем данные пользователя.
	if err := utils.ValidateUserForRegister(user); err != nil {
		s.log.Error("невалидные данные:", err)
		return err
	}

	// Проверяем, существует ли пользователь с такими данными (email или телефон) в базе данных.
	if err := s.repo.TestData(user); err != nil {
		return err
	}

	// Хешируем пароль пользователя для безопасности.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("ошибка хеширования пароля:", err)
		return errInternal
	}
	user.Password = string(passwordHash)

	// Генерируем случайный код подтверждения, который будет отправлен на email.
	code := utils.GenRandCode()

	// Сериализуем данные пользователя в JSON, чтобы сохранить их в Redis.
	userJSON, err := json.Marshal(user)
	if err != nil {
		s.log.Error("не удалось сериализовать данные пользователя:", err)
		return errInternal
	}

	// Сохраняем данные пользователя в Redis с установленным временем жизни (TTL).
	if err = s.client.Set(ctx, code, userJSON, ttl).Err(); err != nil {
		s.log.Error("Ошибка при записи данных в Redis:", err)
		return errInternal
	}

	// Отправляем (ассинхронно) код подтверждения на email пользователя.
	if err = SendConfirmationEmail(user.Email, code); err != nil {
		s.log.Error("Ошибка отправки кода подтверждения на email:", err)
	}

	// Если регистрация прошла успешно, возвращаем nil.
	return nil
}

func (s *AuthService) ConfirmEmail(code string) error {
	// Проверяем, что код подтверждения не пустой.
	if code == "" {
		return errInvalidData
	}

	// Получаем данные из Redis по коду подтверждения.
	result, err := s.client.Get(ctx, code).Result()
	if err == redis.Nil {
		// Если данных нет или срок действия истёк, возвращаем ошибку.
		return errors.New("код подтверждения не найден или срок действия истек")
	}
	if err != nil {
		s.log.Error("Ошибка при получении данных из Redis:", err)
		return errors.New("ошибка проверки кода подтверждения")
	}

	// Декодируем данные пользователя из строки JSON.
	var user entities.User
	if err = json.Unmarshal([]byte(result), &user); err != nil {
		s.log.Error("Ошибка при декодировании данных из Redis:", err)
		return errors.New("не удалось обработать данные пользователя")
	}

	// Сохраняем пользователя в базе данных.
	if err = s.repo.SaveUser(&user); err != nil {
		return err
	}

	// Если всё прошло успешно, возвращаем nil.
	return nil
}

func (s *AuthService) SignIn(user *entities.User, userAddr string) (*entities.TokensResponse, error) {
	// Вытаскиваем данные пользователя по email из репозитория
	resUser, err := s.repo.DBVerifyUser(user)
	if err != nil {
		return nil, errIncorrectPasAndEmail
	}
	resUser.Email = user.Email

	if resUser.Status == "blocked" || resUser.Status == "removed" {
		s.log.Error("Попытка входа заблокированного или удалённого пользователя", nil)
		return nil, errors.New("пользователь удалён или заблокирован")
	}

	// Проверяем соответствие пароля с хешированным паролем в базе данных
	err = bcrypt.CompareHashAndPassword([]byte(resUser.Password), []byte(user.Password))
	if err != nil {
		s.log.Error("Ошибка проверки подленности пароля", err)
		return nil, errIncorrectPasAndEmail
	}

	// Обновляем статус пользователя на 'active', только если пароль верен
	err = s.repo.UpdateRecord("users", map[string]interface{}{
		"status": "active",
	}, resUser.ID)
	if err != nil {
		return nil, err
	}

	// Генерация access токена
	accessToken, err := utils.GenerateAccessToken(resUser, ttlAccess)
	if err != nil {
		s.log.Error("ошибка при генерации access токена:", err)
		return nil, err
	}

	// Генерация refresh токена
	refreshToken, expiresAt, err := utils.GenerateRefreshToken(resUser, ttlRefresh)
	if err != nil {
		s.log.Error("ошибка при генерации refresh токена:", err)
		return nil, err
	}

	// Сохранение данных о токенах в таблицу tokens
	err = s.repo.PersistToken(resUser.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveTCPAddr("tcp", userAddr)
	if err != nil {
		s.log.Error("Ошибка при разборе адреса:", err)
		return nil, errInternal
	}
	ipAddress := addr.IP.String()

	// Сохранение данных о входе в таблицу login_history
	err = s.repo.SaveLoginHistory(resUser.ID, ipAddress)
	if err != nil {
		return nil, err
	}

	// Возвращаем успешный ответ с токенами
	response := &entities.TokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return response, nil
}

func (s *AuthService) SignOut(refreshToken string) error {
	// Разбор refresh токена
	tokenClaim, err := middlewares.ParseRefreshToken(refreshToken)
	if err != nil {
		s.log.Error("ошибка при парсинге refresh токена:", err)
		return err
	}

	// Обновляем статус пользователя на 'active', только если пароль верен
	err = s.repo.UpdateRecord("users", map[string]interface{}{
		"status": "suspended",
	}, tokenClaim.ID)
	if err != nil {
		return err
	}

	// Удаление токена из базы данных
	if err := s.repo.DeleteTokenByID(tokenClaim.ID); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) RefreshTokens(refreshToken string) (*entities.TokensResponse, error) {
	// Валидация refresh токена
	if err := middlewares.ValidateRefreshToken(refreshToken); err != nil {
		s.log.Error("ошибка при валидации refresh токена:", err)
		return nil, err
	}

	// Разбор refresh токена
	tokenClaim, err := middlewares.ParseRefreshToken(refreshToken)
	if err != nil {
		s.log.Error("ошибка при парсинге refresh токена:", err)
		return nil, err
	}

	// Запрос в базу данных для получения времени протухания токена по ID
	tokenInDB, err := s.repo.GetTokenExpiryTime(tokenClaim.ID)
	if err != nil {
		return nil, err
	}

	// Проверка на то, что токен ещё не протух
	if tokenClaim.ExpiresAt == tokenInDB.ExpiresAt {
		return nil, errors.New("время действия токена просрочено")
	}

	// Запрос в базу данных для получения данных пользователя
	user, err := s.repo.GetUserByID(tokenClaim.ID)
	if err != nil {
		return nil, errInternal
	}

	// Генерация access токена
	accessToken, err := utils.GenerateAccessToken(user, ttlAccess)
	if err != nil {
		s.log.Error("ошибка при генерации access токена:", err)
		return nil, err
	}

	// Генерация refresh токена
	refreshToken, expiresAt, err := utils.GenerateRefreshToken(user, ttlRefresh)
	if err != nil {
		s.log.Error("ошибка при генерации refresh токена:", err)
		return nil, err
	}

	// Сохранение данных о токенах в таблицу tokens
	err = s.repo.PersistToken(user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	// Возвращаем успешный ответ с токенами
	response := &entities.TokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return response, nil
}
