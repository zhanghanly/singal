package singal

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"strings"
	"time"
)

type EmailService struct {
	smtpHost  string
	smtpPort  int
	fromEmail string
	password  string
}

var gEmailService *EmailService

func InitEmailService() {
	gEmailService = &EmailService{
		smtpHost:  gConfig.Email.SMTPHost,
		smtpPort:  gConfig.Email.SMTPPort,
		fromEmail: gConfig.Email.FromEmail,
		password:  gConfig.Email.Password,
	}
}

func (s *EmailService) generateCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	return string(code), nil
}

func getVerificationCodeKey(email string) string {
	return fmt.Sprintf("verification_code:%s", strings.ToLower(email))
}

func (s *EmailService) SendVerificationCode(email string) error {
	code, err := s.generateCode()
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	ctx := context.Background()
	redisClient := GetRedisClient()
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	err = redisClient.Set(ctx, getVerificationCodeKey(email), code, 10*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store verification code: %w", err)
	}

	subject := "Your Verification Code"
	body := fmt.Sprintf("Your verification code is: %s\nThis code will expire in 10 minutes.", code)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.fromEmail, s.password, s.smtpHost)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.fromEmail, to, subject, body)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort),
		auth,
		s.fromEmail,
		[]string{to},
		[]byte(msg),
	)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *EmailService) VerifyCode(email, code string) bool {
	ctx := context.Background()
	redisClient := GetRedisClient()
	if redisClient == nil {
		return false
	}

	storedCode, err := redisClient.Get(ctx, getVerificationCodeKey(email)).Result()
	if err != nil {
		return false
	}

	if storedCode == code {
		redisClient.Del(ctx, getVerificationCodeKey(email))
		return true
	}

	return false
}

func GetEmailService() *EmailService {
	return gEmailService
}
