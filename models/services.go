package models

import (
	"encoding/json"
	"fmt"
	"time"

	"example.com/db"
)

type MediaItem struct {
	PublicID string `json:"public_id"`
	FileName string `json:"fileName"`
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
}

type Service struct {
	ID          int64       `json:"id"`
	Name        string      `binding:"required" json:"name"`
	Description string      `json:"description"`
	Price       int64       `binding:"required" json:"price"`
	Currency    string      `binding:"required" json:"currency"`
	Duration    int64       `json:"duration"`
	Timestamp   *time.Time  `json:"timestamp,omitempty"`
	UserID      int64       `json:"user_id"`
	Media       []MediaItem `json:"media"`
}

func GetServicesForUser(id int64) ([]Service, error) {
	query := "SELECT id, name, description, price, duration, media, currency, timestamp FROM services WHERE user_id = $1"
	rows, err := db.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []Service

	for rows.Next() {
		var service Service
		service.Media = []MediaItem{}
		var mediaJson *string
		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Price,
			&service.Duration,
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
			service.Media = []MediaItem{}
		}

		services = append(services, service)
	}

	return services, nil
}

func GetServiceById(id, userId int64) (*Service, error) {
	query := "SELECT id, name, description, price, duration, media, currency, timestamp, user_id FROM services WHERE user_id = $1 AND id = $2"
	row := db.DB.QueryRow(query, userId, id)

	var service Service
	var mediaJson *string
	err := row.Scan(&service.ID, &service.Name, &service.Description, &service.Price, &service.Duration, &mediaJson, &service.Currency, &service.Timestamp, &service.UserID)
	if err != nil {
		return nil, err
	}
	if mediaJson != nil && *mediaJson != "" {
		err = json.Unmarshal([]byte(*mediaJson), &service.Media)
		if err != nil {
			return nil, err
		}
	} else {
		service.Media = []MediaItem{}
	}

	return &service, nil
}

func (s *Service) CreateService() (*Service, error) {
	mediaJson, err := json.Marshal(s.Media)
	if err != nil {
		return nil, err
	}
	query := `
		INSERT INTO services(name, description, price, currency, duration, timestamp, user_id, media)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err = db.DB.QueryRow(
		query,
		s.Name,
		s.Description,
		s.Price,
		s.Currency,
		s.Duration,
		s.Timestamp,
		s.UserID,
		string(mediaJson),
	).Scan(&s.ID)

	if err != nil {
		return nil, err
	}

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

func (s *Service) SaveMedia() (*Service, error) {
	mediaJSON, err := json.Marshal(s.Media)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal media: %w", err)
	}
	query := `
		UPDATE services
		SET media = $1, timestamp = $2
		WHERE id = $3 AND user_id = $4
	`

	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	now := time.Now().UTC()
	s.Timestamp = &now

	_, err = stmt.Exec(mediaJSON, s.Timestamp, s.ID, s.UserID)
	return s, err
}

func (s *Service) DeleteService() error {
	query := `DELETE FROM services WHERE id = $1 AND user_id = $2`

	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()

	result, err := stmt.Exec(s.ID, s.UserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("service not found or you are not authorized to delete it")
	}

	return nil
}
