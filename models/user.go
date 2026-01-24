package models

import (
	"errors"

	"example.com/db"
	"example.com/utils"
)

type User struct {
	ID       int64
	Email    string `binding:"required"`
	Password string `binding:"required"`
	Alias    string
}

func (u *User) Save() error {
	query := `INSERT INTO users(email, password) VALUES ($1, $2) RETURNING id`

	hashedPassword, err := utils.Hash(u.Password)
	if err != nil {
		return err
	}

	err = db.DB.QueryRow(query, u.Email, hashedPassword).Scan(&u.ID)
	return err
}

func (u *User) ValidateCredentials() error {
	query := `SELECT id, password FROM users WHERE email = $1`
	row := db.DB.QueryRow(query, u.Email)

	var retrievedPassword string
	err := row.Scan(&u.ID, &retrievedPassword)

	if err != nil {
		return err
	}

	passwordIsValid := utils.CheckPasswordHash(u.Password, retrievedPassword)

	if !passwordIsValid {
		return errors.New("invalid credentials")
	}

	return nil
}

func GetUserByAlias(alias string) (*User, error) {
	query := `SELECT id, email, alias FROM users WHERE alias = $1`
	row := db.DB.QueryRow(query, alias)

	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Alias)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetAlias(userId int64) (string, error) {
	query := `SELECT alias FROM users WHERE id = $1`
	var alias string
	err := db.DB.QueryRow(query, userId).Scan(&alias)
	return alias, err
}

func UpdateAlias(userId int64, alias string) error {
	query := `UPDATE users SET alias = $1 WHERE id = $2`
	result, err := db.DB.Exec(query, alias, userId)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}
