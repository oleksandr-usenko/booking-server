package models

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"example.com/db"
)

const timeLayout = "15:04" // "HH:MM"

// API payload
type TimeRangePayload struct {
	Start string `json:"start"` // "HH:MM"
	End   string `json:"end"`   // "HH:MM"
}

// DB/API output per range (for a single day)
type TimeRange struct {
	ID        int64  `json:"id"`
	StartTime string `json:"start"` // "HH:MM"
	EndTime   string `json:"end"`   // "HH:MM"
}

// -------- Single day getter --------
func GetSchedule(ctx context.Context, userID int64, date time.Time) ([]TimeRange, error) {
	day := date.UTC().Format("2006-01-02")
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedules
		WHERE user_id = $1 AND date = $2::date
		ORDER BY start_time
	`, userID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TimeRange
	for rows.Next() {
		var tr TimeRange
		if err := rows.Scan(&tr.ID, &tr.StartTime, &tr.EndTime); err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

// -------- Range getter (today + next N-1 days) --------

type ScheduleByDate map[string][]TimeRange // key: "YYYY-MM-DD"

func GetScheduleForRange(ctx context.Context, userID int64, start time.Time, days int) (ScheduleByDate, error) {
	if days <= 0 {
		days = 1
	}

	// half-open range: [start, end)
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endDay := startDay.AddDate(0, 0, days)

	rows, err := db.DB.QueryContext(ctx, `
		SELECT date, id, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedules
		WHERE user_id = $1
		  AND date >= $2::date
		  AND date <  $3::date
		ORDER BY date, start_time
	`, userID, startDay, endDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(ScheduleByDate)
	for rows.Next() {
		var d time.Time
		var tr TimeRange
		if err := rows.Scan(&d, &tr.ID, &tr.StartTime, &tr.EndTime); err != nil {
			return nil, err
		}
		key := d.Format("2006-01-02")
		out[key] = append(out[key], tr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Ensure dates with no rows still appear if you want (optional):
	// for i := 0; i < days; i++ {
	//   k := startDay.AddDate(0,0,i).Format("2006-01-02")
	//   if _, ok := out[k]; !ok { out[k] = []TimeRange{} }
	// }

	return out, nil
}

// -------- Save (replace a day) --------

func SaveSchedule(ctx context.Context, userID int64, date time.Time, ranges []TimeRangePayload) ([]TimeRange, error) {
	// Normalize + validate
	norm, err := normalizeAndValidate(ranges)
	if err != nil {
		return nil, err
	}

	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	day := date.UTC().Format("2006-01-02")

	// delete the day
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM schedules WHERE user_id = $1 AND date = $2::date`,
		userID, day,
	); err != nil {
		return nil, err
	}

	// empty payload → commit
	if len(norm) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return []TimeRange{}, nil
	}

	// insert new rows
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO schedules (user_id, date, start_time, end_time)
		VALUES ($1, $2::date, $3::time, $4::time)
		RETURNING id, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	inserted := make([]TimeRange, 0, len(norm))
	for _, r := range norm {
		var tr TimeRange
		if err := stmt.QueryRowContext(ctx, userID, day, r.Start, r.End).
			Scan(&tr.ID, &tr.StartTime, &tr.EndTime); err != nil {
			return nil, err
		}
		inserted = append(inserted, tr)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inserted, nil
}

// -------- Validation helpers --------

type normRange struct {
	Start string
	End   string
}

func normalizeAndValidate(in []TimeRangePayload) ([]normRange, error) {
	if len(in) == 0 {
		return []normRange{}, nil
	}

	type point struct {
		startMin int
		endMin   int
		startStr string
		endStr   string
	}

	pts := make([]point, 0, len(in))

	for _, r := range in {
		startT, err1 := time.Parse(timeLayout, r.Start)
		endT, err2 := time.Parse(timeLayout, r.End)
		if err1 != nil || err2 != nil {
			return nil, errors.New("invalid time format (expected HH:MM)")
		}
		if !startT.Before(endT) {
			return nil, errors.New("each range must satisfy start < end")
		}
		pts = append(pts, point{
			startMin: startT.Hour()*60 + startT.Minute(),
			endMin:   endT.Hour()*60 + endT.Minute(),
			startStr: r.Start,
			endStr:   r.End,
		})
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].startMin < pts[j].startMin })
	for i := 1; i < len(pts); i++ {
		if pts[i].startMin < pts[i-1].endMin {
			return nil, errors.New("time ranges overlap")
		}
	}

	out := make([]normRange, 0, len(pts))
	for _, p := range pts {
		out = append(out, normRange{Start: p.startStr, End: p.endStr})
	}
	return out, nil
}
