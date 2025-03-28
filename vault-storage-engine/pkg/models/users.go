package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
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

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	query := "SELECT id, email, password FROM users WHERE email = ?"
	row := db.QueryRow(query, email)

	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.Password); err != nil {
		log.Printf("failed to get username, %v", err)
		return nil, err
	}

	return &user, nil
}

func CreateUser(db *sql.DB, user *User) error {
	time := time.Now().Format(time.RFC3339)
	query := "INSERT INTO users (username, email, password, created_at) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(query, user.Username, user.Email, user.Password, time)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
