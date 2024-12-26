package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq" // Подключаем драйвер PostgreSQL
	"os"
)

// ConnectPsql - функция для подключения к базе данных
func ConnectPsql() (*sql.DB, error) {
	// Получаем параметры подключения из переменных окружения
	hostDB := os.Getenv("DB_HOST")
	portDB := os.Getenv("DB_PORT")
	userDB := os.Getenv("DB_USER")
	passwordDB := os.Getenv("DB_PASS")
	nameDB := os.Getenv("DB_NAME")

	// Формируем строку подключения
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		hostDB, portDB, userDB, passwordDB, nameDB,
	)

	// Открываем подключение
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}

	// Проверяем подключение
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось запинговать базу данных: %v", err)
	}

	return db, nil
}
