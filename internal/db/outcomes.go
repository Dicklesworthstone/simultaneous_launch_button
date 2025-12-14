package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrOutcomeNotFound indicates the requested outcome does not exist.
var ErrOutcomeNotFound = errors.New("outcome not found")

// ExecutionOutcome represents the result and feedback for an executed request.
type ExecutionOutcome struct {
	// ID is the unique outcome identifier (auto-generated).
	ID int64 `json:"id"`
	// RequestID is the request this outcome belongs to.
	RequestID string `json:"request_id"`
	// Result is a short description of the result (legacy field).
	Result string `json:"result,omitempty"`
	// Notes contains additional notes (legacy field).
	Notes string `json:"notes,omitempty"`
	// CausedProblems indicates if the execution caused issues.
	CausedProblems bool `json:"caused_problems"`
	// ProblemDescription describes the problems encountered.
	ProblemDescription string `json:"problem_description,omitempty"`
	// HumanRating is the human's rating (1-5 scale).
	HumanRating *int `json:"human_rating,omitempty"`
	// HumanNotes contains human feedback.
	HumanNotes string `json:"human_notes,omitempty"`
	// CreatedAt is when the outcome was recorded.
	CreatedAt time.Time `json:"created_at"`
}

// CreateOutcome inserts an execution outcome record.
func (db *DB) CreateOutcome(o *ExecutionOutcome) error {
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}

	result, err := db.Exec(`
		INSERT INTO execution_outcomes (
			request_id, result, notes, caused_problems, problem_description,
			human_rating, human_notes, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		o.RequestID, nullString(o.Result), nullString(o.Notes),
		boolToInt(o.CausedProblems), nullString(o.ProblemDescription),
		nullInt(o.HumanRating), nullString(o.HumanNotes),
		o.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("creating outcome: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting outcome id: %w", err)
	}
	o.ID = id
	return nil
}

// GetOutcome retrieves an outcome by ID.
func (db *DB) GetOutcome(id int64) (*ExecutionOutcome, error) {
	row := db.QueryRow(`
		SELECT id, request_id, result, notes, caused_problems, problem_description,
		       human_rating, human_notes, created_at
		FROM execution_outcomes WHERE id = ?
	`, id)
	return scanOutcomeRow(row)
}

// GetOutcomeForRequest retrieves the outcome for a request.
func (db *DB) GetOutcomeForRequest(requestID string) (*ExecutionOutcome, error) {
	row := db.QueryRow(`
		SELECT id, request_id, result, notes, caused_problems, problem_description,
		       human_rating, human_notes, created_at
		FROM execution_outcomes WHERE request_id = ?
		ORDER BY created_at DESC LIMIT 1
	`, requestID)
	return scanOutcomeRow(row)
}

// ListOutcomes returns all outcomes, most recent first.
func (db *DB) ListOutcomes(limit int) ([]*ExecutionOutcome, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, request_id, result, notes, caused_problems, problem_description,
		       human_rating, human_notes, created_at
		FROM execution_outcomes
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing outcomes: %w", err)
	}
	defer rows.Close()
	return scanOutcomeList(rows)
}

// ListProblematicOutcomes returns outcomes where caused_problems is true.
func (db *DB) ListProblematicOutcomes(limit int) ([]*ExecutionOutcome, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, request_id, result, notes, caused_problems, problem_description,
		       human_rating, human_notes, created_at
		FROM execution_outcomes
		WHERE caused_problems = 1
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing problematic outcomes: %w", err)
	}
	defer rows.Close()
	return scanOutcomeList(rows)
}

// UpdateOutcome updates an existing outcome.
func (db *DB) UpdateOutcome(o *ExecutionOutcome) error {
	result, err := db.Exec(`
		UPDATE execution_outcomes SET
			result = ?,
			notes = ?,
			caused_problems = ?,
			problem_description = ?,
			human_rating = ?,
			human_notes = ?
		WHERE id = ?
	`,
		nullString(o.Result), nullString(o.Notes),
		boolToInt(o.CausedProblems), nullString(o.ProblemDescription),
		nullInt(o.HumanRating), nullString(o.HumanNotes),
		o.ID,
	)
	if err != nil {
		return fmt.Errorf("updating outcome: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrOutcomeNotFound
	}
	return nil
}

// RecordOutcome creates or updates an outcome for a request.
func (db *DB) RecordOutcome(requestID string, causedProblems bool, description string, rating *int, notes string) (*ExecutionOutcome, error) {
	// Check if outcome exists
	existing, err := db.GetOutcomeForRequest(requestID)
	if err != nil && !errors.Is(err, ErrOutcomeNotFound) {
		return nil, err
	}

	if existing != nil {
		// Update existing
		existing.CausedProblems = causedProblems
		existing.ProblemDescription = description
		existing.HumanRating = rating
		existing.HumanNotes = notes
		if err := db.UpdateOutcome(existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new
	outcome := &ExecutionOutcome{
		RequestID:          requestID,
		CausedProblems:     causedProblems,
		ProblemDescription: description,
		HumanRating:        rating,
		HumanNotes:         notes,
	}
	if err := db.CreateOutcome(outcome); err != nil {
		return nil, err
	}
	return outcome, nil
}

// OutcomeStats contains aggregate statistics about execution outcomes.
type OutcomeStats struct {
	TotalOutcomes      int     `json:"total_outcomes"`
	ProblematicCount   int     `json:"problematic_count"`
	ProblematicPercent float64 `json:"problematic_percent"`
	AvgHumanRating     float64 `json:"avg_human_rating"`
	RatedCount         int     `json:"rated_count"`
}

// GetOutcomeStats returns aggregate statistics for outcomes.
func (db *DB) GetOutcomeStats() (*OutcomeStats, error) {
	stats := &OutcomeStats{}

	// Total count
	if err := db.QueryRow(`SELECT COUNT(*) FROM execution_outcomes`).Scan(&stats.TotalOutcomes); err != nil {
		return nil, fmt.Errorf("counting outcomes: %w", err)
	}

	if stats.TotalOutcomes == 0 {
		return stats, nil
	}

	// Problematic count
	if err := db.QueryRow(`SELECT COUNT(*) FROM execution_outcomes WHERE caused_problems = 1`).Scan(&stats.ProblematicCount); err != nil {
		return nil, fmt.Errorf("counting problematic: %w", err)
	}
	stats.ProblematicPercent = float64(stats.ProblematicCount) / float64(stats.TotalOutcomes) * 100

	// Average rating (only for rated outcomes)
	var avgRating sql.NullFloat64
	if err := db.QueryRow(`
		SELECT AVG(CAST(human_rating AS REAL)), COUNT(*)
		FROM execution_outcomes WHERE human_rating IS NOT NULL
	`).Scan(&avgRating, &stats.RatedCount); err != nil {
		return nil, fmt.Errorf("calculating avg rating: %w", err)
	}
	if avgRating.Valid {
		stats.AvgHumanRating = avgRating.Float64
	}

	return stats, nil
}

// RequestStats contains statistics about a specific request's history.
type RequestStats struct {
	TotalRequests  int     `json:"total_requests"`
	ApprovedCount  int     `json:"approved_count"`
	RejectedCount  int     `json:"rejected_count"`
	ExecutedCount  int     `json:"executed_count"`
	ProblematicPct float64 `json:"problematic_pct"`
}

// GetRequestStatsByAgent returns request statistics for a specific agent.
func (db *DB) GetRequestStatsByAgent(agentName string) (*RequestStats, error) {
	stats := &RequestStats{}

	// Total requests by agent
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM requests WHERE requestor_agent = ?
	`, agentName).Scan(&stats.TotalRequests); err != nil {
		return nil, fmt.Errorf("counting requests: %w", err)
	}

	if stats.TotalRequests == 0 {
		return stats, nil
	}

	// Approved/Rejected/Executed counts
	if err := db.QueryRow(`
		SELECT
			SUM(CASE WHEN status IN ('approved', 'executing', 'executed', 'execution_failed') THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'executed' THEN 1 ELSE 0 END)
		FROM requests WHERE requestor_agent = ?
	`, agentName).Scan(&stats.ApprovedCount, &stats.RejectedCount, &stats.ExecutedCount); err != nil {
		return nil, fmt.Errorf("counting by status: %w", err)
	}

	// Problematic percentage
	if stats.ExecutedCount > 0 {
		var problematic int
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM execution_outcomes o
			JOIN requests r ON o.request_id = r.id
			WHERE r.requestor_agent = ? AND o.caused_problems = 1
		`, agentName).Scan(&problematic); err != nil {
			return nil, fmt.Errorf("counting problematic: %w", err)
		}
		stats.ProblematicPct = float64(problematic) / float64(stats.ExecutedCount) * 100
	}

	return stats, nil
}

// TimeToApprovalStats contains statistics about approval times.
type TimeToApprovalStats struct {
	AvgMinutes    float64 `json:"avg_minutes"`
	MedianMinutes float64 `json:"median_minutes"`
	MinMinutes    float64 `json:"min_minutes"`
	MaxMinutes    float64 `json:"max_minutes"`
	SampleSize    int     `json:"sample_size"`
}

// GetTimeToApprovalStats returns statistics about how long it takes for requests to get approved.
// Uses the first approval review's timestamp as the "approved at" time.
func (db *DB) GetTimeToApprovalStats() (*TimeToApprovalStats, error) {
	stats := &TimeToApprovalStats{}

	// Get all approval times by looking at when the first approval review was created
	rows, err := db.Query(`
		SELECT
			(julianday(MIN(rv.created_at)) - julianday(r.created_at)) * 24 * 60 as minutes
		FROM requests r
		JOIN reviews rv ON rv.request_id = r.id AND rv.decision = 'approve'
		WHERE r.status IN ('approved', 'executing', 'executed', 'execution_failed')
		GROUP BY r.id
		HAVING minutes IS NOT NULL
		ORDER BY minutes
	`)
	if err != nil {
		return nil, fmt.Errorf("querying approval times: %w", err)
	}
	defer rows.Close()

	var times []float64
	for rows.Next() {
		var minutes float64
		if err := rows.Scan(&minutes); err != nil {
			return nil, fmt.Errorf("scanning time: %w", err)
		}
		times = append(times, minutes)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.SampleSize = len(times)
	if stats.SampleSize == 0 {
		return stats, nil
	}

	// Calculate stats
	var sum float64
	stats.MinMinutes = times[0]
	stats.MaxMinutes = times[0]
	for _, t := range times {
		sum += t
		if t < stats.MinMinutes {
			stats.MinMinutes = t
		}
		if t > stats.MaxMinutes {
			stats.MaxMinutes = t
		}
	}
	stats.AvgMinutes = sum / float64(stats.SampleSize)

	// Median
	mid := stats.SampleSize / 2
	if stats.SampleSize%2 == 0 {
		stats.MedianMinutes = (times[mid-1] + times[mid]) / 2
	} else {
		stats.MedianMinutes = times[mid]
	}

	return stats, nil
}

func scanOutcomeRow(row *sql.Row) (*ExecutionOutcome, error) {
	o := &ExecutionOutcome{}
	var result, notes, problemDesc, humanNotes sql.NullString
	var humanRating sql.NullInt64
	var causedProblems int
	var created string

	err := row.Scan(&o.ID, &o.RequestID, &result, &notes, &causedProblems,
		&problemDesc, &humanRating, &humanNotes, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOutcomeNotFound
		}
		return nil, fmt.Errorf("scanning outcome: %w", err)
	}

	o.Result = result.String
	o.Notes = notes.String
	o.CausedProblems = causedProblems != 0
	o.ProblemDescription = problemDesc.String
	if humanRating.Valid {
		r := int(humanRating.Int64)
		o.HumanRating = &r
	}
	o.HumanNotes = humanNotes.String
	o.CreatedAt, _ = time.Parse(time.RFC3339, created)

	return o, nil
}

func scanOutcomeList(rows *sql.Rows) ([]*ExecutionOutcome, error) {
	var list []*ExecutionOutcome
	for rows.Next() {
		o := &ExecutionOutcome{}
		var result, notes, problemDesc, humanNotes sql.NullString
		var humanRating sql.NullInt64
		var causedProblems int
		var created string

		if err := rows.Scan(&o.ID, &o.RequestID, &result, &notes, &causedProblems,
			&problemDesc, &humanRating, &humanNotes, &created); err != nil {
			return nil, fmt.Errorf("scanning outcomes: %w", err)
		}

		o.Result = result.String
		o.Notes = notes.String
		o.CausedProblems = causedProblems != 0
		o.ProblemDescription = problemDesc.String
		if humanRating.Valid {
			r := int(humanRating.Int64)
			o.HumanRating = &r
		}
		o.HumanNotes = humanNotes.String
		o.CreatedAt, _ = time.Parse(time.RFC3339, created)

		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// nullInt converts *int to sql.NullInt64
func nullInt(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}
