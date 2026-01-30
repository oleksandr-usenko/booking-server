package models

import (
	"database/sql"
	"errors"

	"example.com/db"
	"example.com/utils"
)

type User struct {
	ID              int64
	Email           string  `binding:"required"`
	Password        string  `binding:"required"`
	Alias           string
	Name            *string
	OAuthProvider   *string
	OAuthProviderID *string
}

type OAuthUser struct {
	Email           string
	Name            string
	OAuthProvider   string
	OAuthProviderID string
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

// FindOrCreateOAuthUser finds an existing OAuth user or creates a new one
func FindOrCreateOAuthUser(oauthUser *OAuthUser) (*User, error) {
	// First, try to find by OAuth provider and provider ID
	query := `SELECT id, email, alias, name FROM users WHERE oauth_provider = $1 AND oauth_provider_id = $2`
	row := db.DB.QueryRow(query, oauthUser.OAuthProvider, oauthUser.OAuthProviderID)

	var user User
	var name sql.NullString
	err := row.Scan(&user.ID, &user.Email, &user.Alias, &name)
	if err == nil {
		if name.Valid {
			user.Name = &name.String
		}
		return &user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Check if a user with this email already exists (linked to another provider or password auth)
	emailQuery := `SELECT id, email, alias, name, oauth_provider FROM users WHERE email = $1`
	row = db.DB.QueryRow(emailQuery, oauthUser.Email)

	var existingProvider sql.NullString
	err = row.Scan(&user.ID, &user.Email, &user.Alias, &name, &existingProvider)
	if err == nil {
		// User exists with this email - update to link OAuth provider
		updateQuery := `UPDATE users SET oauth_provider = $1, oauth_provider_id = $2, name = COALESCE(name, $3) WHERE id = $4`
		_, err = db.DB.Exec(updateQuery, oauthUser.OAuthProvider, oauthUser.OAuthProviderID, oauthUser.Name, user.ID)
		if err != nil {
			return nil, err
		}
		if name.Valid {
			user.Name = &name.String
		}
		return &user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	insertQuery := `
		INSERT INTO users (email, name, oauth_provider, oauth_provider_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, alias
	`
	err = db.DB.QueryRow(insertQuery, oauthUser.Email, oauthUser.Name, oauthUser.OAuthProvider, oauthUser.OAuthProviderID).Scan(&user.ID, &user.Alias)
	if err != nil {
		return nil, err
	}

	user.Email = oauthUser.Email
	user.Name = &oauthUser.Name

	return &user, nil
}
