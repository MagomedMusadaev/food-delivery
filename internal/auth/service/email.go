package service

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// SendConfirmationEmail отправляет электронное письмо с кодом подтверждения на указанный адрес.
// to - адрес электронной почты получателя, code - код подтверждения, который будет отправлен в письме.
func SendConfirmationEmail(to, code string) error {

	from := os.Getenv("MAIL_FROM")         // Адрес электронной почты отправителя
	password := os.Getenv("MAIL_PASSWORD") // Пароль для SMTP-сервера
	smtpHost := os.Getenv("MAIL_HOST")     // Хост SMTP-сервера
	smtpPort := os.Getenv("MAIL_PORT")     // Порт SMTP-сервера

	// Формируем сообщение, включая заголовок и тело письма
	var builder strings.Builder
	builder.WriteString("Subject: Подтверждение регистрации\n\n")
	builder.WriteString(fmt.Sprintf("Ваш код подтверждения: %s", code))

	msg := []byte(builder.String())

	// Настраиваем аутентификацию для отправки почты
	auth := smtp.PlainAuth("", from, password, smtpHost)

	go func() {
		// Отправляем почту с использованием smtp.SendMail
		err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
		if err != nil { //TODO: надо доработать логику (пока так)
			err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
		}
	}()

	return nil // Возвращаем nil, если отправка прошла успешно
}
