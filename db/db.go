package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	// local development DSN
	username := os.Getenv("PG_USERNAME")
	password := os.Getenv("PG_PASSWORD")
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:5432/mydb?sslmode=disable", username, password)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		panic("Could not connect to DB: " + err.Error())
	}

	// Optional connection pool
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)

	createTables()
}

func createTables() {
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id BIGSERIAL PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL
    );`

	createEventsTable := `
    CREATE TABLE IF NOT EXISTS events (
        id BIGSERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        description TEXT NOT NULL,
        location TEXT NOT NULL,
        dateTime TIMESTAMP NOT NULL,
        user_id BIGINT REFERENCES users(id)
    );`

	createRegistrationTable := `
    CREATE TABLE IF NOT EXISTS registrations (
        id BIGSERIAL PRIMARY KEY,
        event_id BIGINT REFERENCES events(id),
        user_id BIGINT REFERENCES users(id)
    );`

	createServicesTable := `
    CREATE TABLE IF NOT EXISTS services (
        id BIGSERIAL PRIMARY KEY,
        name TEXT,
        description TEXT,
        price BIGINT,
        duration BIGINT,
        user_id BIGINT REFERENCES users(id),
        media JSONB,
        currency TEXT
    );`

	statements := []string{
		createUsersTable,
		createEventsTable,
		createServicesTable,
		createRegistrationTable,
	}

	for _, stmt := range statements {
		_, err := DB.Exec(stmt)
		if err != nil {
			panic("Could not create table: " + err.Error())
		}
	}

	fmt.Println("PostgreSQL tables created successfully!")
}
