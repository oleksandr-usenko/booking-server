package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var dsn string

	dsn = os.Getenv("DATABASE_URL")

	if dsn == "" {
		username := os.Getenv("PG_USERNAME")
		password := os.Getenv("PG_PASSWORD")
		dsn = fmt.Sprintf("postgres://%s:%s@localhost:5432/mydb?sslmode=disable", username, password)
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		panic("Could not connect to DB: " + err.Error())
	}

	// Optional connection pool
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection with retries
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		err = DB.Ping()
		if err == nil {
			log.Println("Successfully connected to database")
			break
		}
		log.Printf("Failed to ping database (attempt %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		panic("Could not verify DB connection after retries: " + err.Error())
	}

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
        currency TEXT,
        timestamp TIMESTAMP
    );`

	createSchedulesTable := `
		CREATE TABLE IF NOT EXISTS schedules (
			id         BIGSERIAL PRIMARY KEY,
			user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			date       DATE   NOT NULL,
			start_time TIME   NOT NULL,
			end_time   TIME   NOT NULL,
			CONSTRAINT chk_time_range CHECK (end_time > start_time)
		);

		CREATE INDEX IF NOT EXISTS idx_schedules_user_date
		ON schedules (user_id, date);
	`

	createAppointmentsTable := `
		CREATE TABLE IF NOT EXISTS appointments (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			service_id BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
			date       DATE NOT NULL,
			start_time TIME NOT NULL,
			end_time   TIME NOT NULL,
			first_name TEXT NOT NULL,
			last_name  TEXT NOT NULL,
			email      TEXT NOT NULL,
			phone      TEXT NOT NULL,
			instagram  TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_appointments_user_id
		ON appointments (user_id);
	`

	statements := []string{
		createUsersTable,
		createEventsTable,
		createServicesTable,
		createRegistrationTable,
		createSchedulesTable,
		createAppointmentsTable,
	}

	for _, stmt := range statements {
		_, err := DB.Exec(stmt)
		if err != nil {
			panic("Could not create table: " + err.Error())
		}
	}

	// Add timestamp column if it doesn't exist (for existing databases)
	alterServicesTable := `
		ALTER TABLE services
		ADD COLUMN IF NOT EXISTS timestamp TIMESTAMP;
	`
	_, err := DB.Exec(alterServicesTable)
	if err != nil {
		panic("Could not alter services table: " + err.Error())
	}

	// Add alias column to users table (for existing databases)
	alterUsersTable := `
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS alias TEXT UNIQUE DEFAULT gen_random_uuid()::text;
	`
	_, err = DB.Exec(alterUsersTable)
	if err != nil {
		panic("Could not alter users table: " + err.Error())
	}

	// Backfill alias for existing users that have NULL
	_, err = DB.Exec(`UPDATE users SET alias = gen_random_uuid()::text WHERE alias IS NULL`)
	if err != nil {
		panic("Could not backfill user aliases: " + err.Error())
	}

	fmt.Println("PostgreSQL tables created successfully!")
}
