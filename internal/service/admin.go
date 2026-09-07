package service

import (
	"context"
	"errors"

	"friends-records/internal/models/mysql"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct{ Store *mysql.Store }

func (s Admin) Login(ctx context.Context, username, password string) (mysql.AdminUser, error) {
	user, err := s.Store.AdminByUsername(ctx, username)
	if err != nil {
		return mysql.AdminUser{}, err
	}
	if !user.Enabled || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return mysql.AdminUser{}, mysql.ErrInvalidCredentials
	}
	return user, nil
}
func IsInvalidCredentials(err error) bool { return errors.Is(err, mysql.ErrInvalidCredentials) }
