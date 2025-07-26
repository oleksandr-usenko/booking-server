package models

import "example.com/db"

type Service struct {
	ID          int64
	Name        string `binding:"required"`
	Description string
	Price       int `binding:"required"`
	Duration    int
	UserID      int64
}

func GetServicesForUser(id int64) (*Service, error) {
	query := "SELECT * FROM services WHERE id = ?"
	row := db.DB.QueryRow(query, id)

	var service Service
	err := row.Scan(&service.ID, &service.Name, &service.Description, &service.Price, &service.Duration, &service.UserID)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (s *Service) CreateService() (*Service, error) {
	query := `
		INSERT INTO services(name, description, price, duration, user_id)
		VALUES (?, ?, ?, ?, ?)
	`
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(s.Name, s.Description, s.Price, s.Duration, s.UserID)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	s.ID = id
	return s, err
}
