package models

import (
	"database/sql"
	"log"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

// Hashes the password with bycrpt for database injection
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to get password")
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	query := "SELECT id, username, password FROM users WHERE username = ?"
	row := db.QueryRow(query, username)

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.Password); err != nil {
		log.Fatal("failed to get username")
		return nil, err
	}

	return &user, nil
}

func CreateUser(db *sql.DB, user *User) error {
	query := "INSERT INTO users (username, password) VALUES (?, ?)"
	_, err := db.Exec(query, user.Username, user.Password)
	if err != nil {
		return err
	}

	return nil
}
