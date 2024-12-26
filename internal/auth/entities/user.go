package entities

import "time"

// User представляет структуру данных пользователя
type User struct {
	ID        int       `json:"id" db:"id"`                  // Уникальный идентификатор пользователя
	Firstname string    `json:"firstname" db:"firstname"`    // Имя пользователя
	Email     string    `json:"email" db:"email"`            // Электронная почта пользователя
	Password  string    `json:"password" db:"password_hash"` // Хэшированный пароль пользователя (не показывается в JSON)
	Phone     string    `json:"phone" db:"phone"`            // Телефон пользователя
	CreatedAt time.Time `json:"createdAt" db:"created_at"`   // Дата и время создания пользователя
	Status    string    `json:"status" db:"status"`          // Статус пользователя (например: active, suspended, blocked, removed)
	Role      string    `json:"role" db:"role"`              // Роль пользователя (например: user, admin)
}

// NewUser создает новый объект пользователя с заданными значениями
func NewUser(firstname, email, password, phone string) *User {
	return &User{
		Firstname: firstname, // Имя пользователя
		Email:     email,     // Электронная почта
		Password:  password,  // Хэш пароля
		Phone:     phone,     // Телефон
		//CreatedAt: time.Now(), // Текущее по умолчанию
		//Status:    "active",   // Статус по умолчанию
		//Role:      "user",     // Роль по умолчанию
	}
}
