package models

import (
	"encoding/json"
	"fmt"

	"example.com/db"
)

type Service struct {
	ID          int64    `json:"id"`
	Name        string   `binding:"required" json:"name"`
	Description string   `json:"description"`
	Price       int64    `binding:"required" json:"price"`
	Duration    int64    `json:"duration"`
	UserID      int64    `json:"user_id"`
	Media       []string `json:"media_urls"`
}

func GetServicesForUser(id int64) ([]Service, error) {
	query := "SELECT * FROM services WHERE user_id = ?"
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

func (s *Service) CreateService() (*Service, error) {
	mediaJson, err := json.Marshal(s.Media)
	if err != nil {
		return nil, err
	}
	query := `
		INSERT INTO services(name, description, price, duration, user_id, media_urls)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(s.Name, s.Description, s.Price, s.Duration, s.UserID, string(mediaJson))
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	s.ID = id
	return s, err
}
