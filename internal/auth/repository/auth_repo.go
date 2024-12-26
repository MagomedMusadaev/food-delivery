package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"food-delivery/internal/auth/entities"
	"food-delivery/pkg/logger"
	"github.com/golang-jwt/jwt/v4"
	"strings"
	"time"
)

var (
	errInternal = errors.New("произошла внутренняя ошибка")
	errNoRows   = errors.New("пользователь с таким email не существует")
	errEmail    = errors.New("пользователь с таким email уже существует")
	errPhone    = errors.New("пользователь с таким номером телефона уже существует")
)

type AuthRepoInt interface {
	TestData(user *entities.User) error
	SaveUser(user *entities.User) error
	DBVerifyUser(user *entities.User) (*entities.User, error)
	PersistToken(userID int, refreshToken string, expiresAt time.Time) error
	GetUserByID(id int) (*entities.User, error)
	GetTokenExpiryTime(userID int) (*entities.RefreshClaim, error)
	DeleteTokenByID(userID int) error
	UpdateRecord(table string, fields map[string]interface{}, id int) error
	SaveLoginHistory(userID int, ipAddress string) error
}

type AuthRepository struct {
	db  *sql.DB
	log *logger.Logger
}

func NewAuthRepository(db *sql.DB, log *logger.Logger) *AuthRepository {
	return &AuthRepository{
		db:  db,
		log: log,
	}
}

func (r *AuthRepository) TestData(user *entities.User) error {
	var temp int

	query := `SELECT 1 FROM users WHERE phone = $1 OR email = $2`

	// Выполняем запрос и пытаемся считать результат
	if err := r.db.QueryRow(query, user.Phone, user.Email).Scan(&temp); err != nil {
		if err == sql.ErrNoRows {
			return nil // Если пользователь с такими данными не найден, возвращаем nil
		}
		// Логирование других ошибок
		r.log.Error("Ошибка при получении данных из DB:", err)
		return errInternal
	}

	// Проверяем, какие именно данные конфликтуют.
	return r.checkConflict(user)
}

// checkConflict проверяет, есть ли конфликт по email или phone.
func (r *AuthRepository) checkConflict(user *entities.User) error {
	var temp int

	// Запрос для проверки наличия пользователя с указанным email.
	queryEmail := `SELECT 1 FROM users WHERE email = $1`
	err := r.db.QueryRow(queryEmail, user.Email).Scan(&temp)
	if err == nil {
		r.log.Error("Ошибка при получении данных из DB:", err)
		return errEmail
	}

	// Запрос для проверки наличия пользователя с указанным phone.
	queryPhone := `SELECT 1 FROM users WHERE phone = $1`
	err = r.db.QueryRow(queryPhone, user.Phone).Scan(&temp)
	if err == nil {
		r.log.Error("Ошибка при получении данных из DB:", err)
		return errPhone
	}

	return nil
}

func (r *AuthRepository) SaveUser(user *entities.User) error {
	query := `INSERT INTO users (firstname, email, password_hash, phone) VALUES ($1, $2, $3, $4)`

	// Выполняем запрос с параметрами.
	_, err := r.db.Exec(query, user.Firstname, user.Email, user.Password, user.Phone)
	if err != nil {
		r.log.Error("Ошибка при сохранении пользователя: ", err)
		return errInternal
	}

	return nil
}

// DBVerifyUser проверяет наличие пользователя в базе данных по его email и вытаскает его данные.
func (r *AuthRepository) DBVerifyUser(user *entities.User) (*entities.User, error) {
	var resUser entities.User

	// Выполняем запрос с email из переданного пользователя
	query := `SELECT id, password_hash, status, role FROM users WHERE email = $1`

	err := r.db.QueryRow(query, user.Email).Scan(&resUser.ID, &resUser.Password, &resUser.Status, &resUser.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		r.log.Error("ошибка получения пользователя:", err)
		return nil, errInternal
	}

	return &resUser, nil
}

func (r *AuthRepository) PersistToken(userID int, refreshToken string, expiresAt time.Time) error {
	// SQL-запрос для вставки данных о токене в таблицу tokens или обновления, если запись уже существует
	query := `
		INSERT INTO tokens (user_id, token, expires_at, created_at) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			token = $2, 
			expires_at = $3, 
			created_at = $4;
	`

	// Выполняем запрос с передачей параметров: userID, refreshToken, expiresAt и текущего времени (created_at).
	_, err := r.db.Exec(query, userID, refreshToken, expiresAt, time.Now())
	if err != nil {
		r.log.Error("Ошибка при сохранении токена в базу данных:", err)
		return errInternal
	}

	// Если все прошло успешно, возвращаем nil.
	return nil
}

func (r *AuthRepository) GetUserByID(userID int) (*entities.User, error) {
	var user entities.User

	query := `SELECT id, email, role FROM users WHERE id = $1`

	// Выполняем запрос в базу данных
	err := r.db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Role)
	if err != nil {
		r.log.Error("Ошибка при получении данных из DB:", err)
		return nil, errors.New("пользователь не найден")
	}

	return &user, nil
}

func (r *AuthRepository) DeleteTokenByID(userID int) error {
	query := `DELETE FROM tokens WHERE user_id = $1`

	// Выполняем запрос в базу данных
	if _, err := r.db.Exec(query, userID); err != nil {
		r.log.Error("ошибка при удалени информации из tokens таблицы:", err)
		return errInternal
	}

	return nil
}

func (r *AuthRepository) GetTokenExpiryTime(userID int) (*entities.RefreshClaim, error) {
	var claim entities.RefreshClaim
	var expiresAt time.Time

	query := `SELECT expires_at FROM tokens WHERE user_id = $1`

	// Выполняем запрос в базу данных и получаем время истечения токена
	err := r.db.QueryRow(query, userID).Scan(&expiresAt)
	if err != nil {
		r.log.Error("Ошибка при получении данных из DB:", err)
		return nil, errors.New("пользователь не найден")
	}

	// Преобразуем время в jwt.NumericDate
	claim.ExpiresAt = jwt.NewNumericDate(expiresAt)

	return &claim, nil
}

// UpdateRecord обновляет поля в указанной таблице на основе данных, переданных в мапе.
func (r *AuthRepository) UpdateRecord(table string, fields map[string]interface{}, id int) error {
	// Строим строку SET для SQL запроса
	setParts := []string{}
	values := []interface{}{}
	paramCount := 1 // Счётчик для параметров

	// Проходим по всем полям и значениям
	for field, value := range fields {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, paramCount)) // $1, $2, ...
		values = append(values, value)
		paramCount++ // Увеличиваем счётчик параметров
	}

	// Строим SQL запрос
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", table, strings.Join(setParts, ", "), paramCount)
	values = append(values, id) // Добавляем id в параметры

	// Выполняем запрос
	_, err := r.db.Exec(query, values...)
	if err != nil {
		r.log.Error("Ошибка при обновлении записи в таблице:", err)
		return errInternal
	}
	return nil
}

func (r *AuthRepository) SaveLoginHistory(userID int, ipAddress string) error {
	query := `INSERT INTO login_history (user_id, ip_address) VALUES ($1, $2)` // время Now() авт.

	// Выполняем запрос
	if _, err := r.db.Exec(query, userID, ipAddress); err != nil {
		r.log.Error("Ошибка при записи данных в таблицу login_history:", err)
		return errInternal
	}

	return nil
}
