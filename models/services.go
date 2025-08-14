package models

import (
	"encoding/json"
	"fmt"
	"time"

	"example.com/db"
)

type Service struct {
	ID          int64      `json:"id"`
	Name        string     `binding:"required" json:"name"`
	Description string     `json:"description"`
	Price       int64      `binding:"required" json:"price"`
	Currency    string     `binding:"required" json:"currency"`
	Duration    int64      `json:"duration"`
	Timestamp   *time.Time `json:"timestamp,omitempty"`
	UserID      int64      `json:"user_id"`
	Media       []string   `json:"media_urls"`
}

func GetServicesForUser(id int64) ([]Service, error) {
	query := "SELECT * FROM services WHERE user_id = $1"
	rows, err := db.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []Service

	for rows.Next() {
		var service Service
		var mediaJson *string
		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Price,
			&service.Duration,
			&service.UserID,
			&mediaJson,
			&service.Currency,
			&service.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		if mediaJson != nil && *mediaJson != "" {
			err = json.Unmarshal([]byte(*mediaJson), &service.Media)
			if err != nil {
				return nil, fmt.Errorf("failed to decode media JSON: %w", err)
			}
		} else {
			service.Media = []string{}
		}

		services = append(services, service)
	}

	return services, nil
}

func GetServiceById(id, userId int64) (*Service, error) {
	query := "SELECT * FROM services WHERE user_id = $1 AND id = $2"
	row := db.DB.QueryRow(query, userId, id)

	var service Service
	var mediaJson string
	err := row.Scan(&service.ID, &service.Name, &service.Description, &service.Price, &service.Duration, &service.UserID, &mediaJson, &service.Currency, &service.Timestamp)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(mediaJson), &service.Media)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (s *Service) CreateService() (*Service, error) {
	mediaJson, err := json.Marshal(s.Media)
	if err != nil {
		return nil, err
	}
	query := `
		INSERT INTO services(name, description, price, currency, duration, timestamp, user_id, media_urls)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(s.Name, s.Description, s.Price, s.Currency, s.Duration, time.Now().UTC(), s.UserID, string(mediaJson))
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	s.ID = id
	return s, err
}

func (s *Service) UpdateService() error {
	query := `
		UPDATE services
		SET name = $1, description = $2, price = $3, currency = $4, duration = $5, timestamp = $6
		WHERE id = $7 AND user_id = $8
	`

	stmt, err := db.DB.Prepare(query)

	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(s.Name, s.Description, s.Price, s.Currency, s.Duration, time.Now().UTC(), s.ID, s.UserID)
	return err
}
