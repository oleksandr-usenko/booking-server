package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"example.com/db"
)

type Appointment struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"userId"`
	ServiceID int64     `json:"serviceId" binding:"required"`
	Date      string    `json:"date" binding:"required"`
	StartTime string    `json:"startTime" binding:"required"`
	EndTime   string    `json:"endTime" binding:"required"`
	FirstName string    `json:"firstName" binding:"required"`
	LastName  string    `json:"lastName" binding:"required"`
	Email     string    `json:"email" binding:"required"`
	Phone     string    `json:"phone" binding:"required"`
	Instagram string    `json:"instagram,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func CreateAppointment(ctx context.Context, appt *Appointment) error {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find the schedule row that fully contains the requested time range
	var schedID int64
	var schedStart, schedEnd string
	err = tx.QueryRowContext(ctx, `
		SELECT id, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedules
		WHERE user_id = $1 AND date = $2::date
		  AND start_time <= $3::time AND end_time >= $4::time
		LIMIT 1
	`, appt.UserID, appt.Date, appt.StartTime, appt.EndTime).Scan(&schedID, &schedStart, &schedEnd)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("no available timeslot for the requested time")
		}
		return err
	}

	// Delete the matching schedule row
	_, err = tx.ExecContext(ctx, `DELETE FROM schedules WHERE id = $1`, schedID)
	if err != nil {
		return err
	}

	// Insert remaining intervals
	if schedStart != appt.StartTime {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO schedules (user_id, date, start_time, end_time)
			VALUES ($1, $2::date, $3::time, $4::time)
		`, appt.UserID, appt.Date, schedStart, appt.StartTime)
		if err != nil {
			return err
		}
	}
	if appt.EndTime != schedEnd {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO schedules (user_id, date, start_time, end_time)
			VALUES ($1, $2::date, $3::time, $4::time)
		`, appt.UserID, appt.Date, appt.EndTime, schedEnd)
		if err != nil {
			return err
		}
	}

	// Insert the appointment
	err = tx.QueryRowContext(ctx, `
		INSERT INTO appointments (user_id, service_id, date, start_time, end_time, first_name, last_name, email, phone, instagram)
		VALUES ($1, $2, $3::date, $4::time, $5::time, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`, appt.UserID, appt.ServiceID, appt.Date, appt.StartTime, appt.EndTime,
		appt.FirstName, appt.LastName, appt.Email, appt.Phone, appt.Instagram,
	).Scan(&appt.ID, &appt.CreatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetAppointments(ctx context.Context, userID int64) ([]Appointment, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, user_id, service_id, date, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       first_name, last_name, email, phone, instagram, created_at
		FROM appointments
		WHERE user_id = $1
		ORDER BY date, start_time
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var a Appointment
		var date time.Time
		var instagram sql.NullString
		err := rows.Scan(&a.ID, &a.UserID, &a.ServiceID, &date, &a.StartTime, &a.EndTime,
			&a.FirstName, &a.LastName, &a.Email, &a.Phone, &instagram, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		a.Date = date.Format("2006-01-02")
		if instagram.Valid {
			a.Instagram = instagram.String
		}
		appointments = append(appointments, a)
	}
	return appointments, rows.Err()
}

func DeleteAppointment(ctx context.Context, appointmentID string, userID int64) error {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Fetch the appointment details
	var date time.Time
	var startTime, endTime string
	err = tx.QueryRowContext(ctx, `
		SELECT date, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM appointments
		WHERE id = $1 AND user_id = $2
	`, appointmentID, userID).Scan(&date, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("appointment not found")
		}
		return err
	}

	// Delete the appointment
	_, err = tx.ExecContext(ctx, `DELETE FROM appointments WHERE id = $1`, appointmentID)
	if err != nil {
		return err
	}

	// Restore the timeslot and merge adjacent intervals
	dateStr := date.Format("2006-01-02")
	err = restoreAndMergeSlot(ctx, tx, userID, dateStr, startTime, endTime)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func restoreAndMergeSlot(ctx context.Context, tx *sql.Tx, userID int64, date, startTime, endTime string) error {
	// Find adjacent schedule rows that touch the restored slot
	// A row is adjacent if its end_time == startTime or its start_time == endTime
	rows, err := tx.QueryContext(ctx, `
		SELECT id, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedules
		WHERE user_id = $1 AND date = $2::date
		  AND (end_time = $3::time OR start_time = $4::time)
	`, userID, date, startTime, endTime)
	if err != nil {
		return err
	}

	mergedStart := startTime
	mergedEnd := endTime
	var idsToDelete []int64

	for rows.Next() {
		var id int64
		var s, e string
		if err := rows.Scan(&id, &s, &e); err != nil {
			rows.Close()
			return err
		}
		idsToDelete = append(idsToDelete, id)
		if s < mergedStart {
			mergedStart = s
		}
		if e > mergedEnd {
			mergedEnd = e
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Delete adjacent rows that will be merged
	for _, id := range idsToDelete {
		_, err = tx.ExecContext(ctx, `DELETE FROM schedules WHERE id = $1`, id)
		if err != nil {
			return err
		}
	}

	// Insert the merged interval
	_, err = tx.ExecContext(ctx, `
		INSERT INTO schedules (user_id, date, start_time, end_time)
		VALUES ($1, $2::date, $3::time, $4::time)
	`, userID, date, mergedStart, mergedEnd)
	return err
}
