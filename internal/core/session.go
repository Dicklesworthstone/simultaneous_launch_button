package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// ErrSessionProgramMismatch indicates an active session exists, but belongs to a different program.
var ErrSessionProgramMismatch = errors.New("active session belongs to a different program")

// SessionSummary is a safe-to-serialize view of a session (excludes session_key).
type SessionSummary struct {
	ID           string
	AgentName    string
	Program      string
	Model        string
	ProjectPath  string
	StartedAt    time.Time
	LastActiveAt time.Time
}

// SessionGCOptions configures stale session garbage collection.
type SessionGCOptions struct {
	ProjectPath string
	Threshold   time.Duration
	DryRun      bool
}

// SessionGCResult reports stale session garbage collection results.
type SessionGCResult struct {
	ProjectPath string
	Threshold   time.Duration
	Cutoff      time.Time
	Sessions    []SessionSummary
	EndedIDs    []string
	SkippedIDs  []string
}

// ResumeOptions configures session resume behavior.
type ResumeOptions struct {
	AgentName        string
	Program          string
	Model            string
	ProjectPath      string
	CreateIfMissing  bool
	ForceEndMismatch bool
}

// ResumeSession resumes an existing active session (agent_name + project_path) or creates a new one.
//
// Behavior:
// - If an active session exists and Program is specified, it must match (unless ForceEndMismatch is true).
// - On successful resume, updates the session heartbeat (last_active_at) and returns the session (with session_key).
// - If no active session exists:
//   - CreateIfMissing=true → creates a new session and returns it
//   - CreateIfMissing=false → returns db.ErrSessionNotFound
func ResumeSession(dbConn *db.DB, opts ResumeOptions) (*db.Session, error) {
	if opts.AgentName == "" {
		return nil, fmt.Errorf("agent_name is required")
	}
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	sess, err := dbConn.GetActiveSession(opts.AgentName, opts.ProjectPath)
	if err != nil {
		if errors.Is(err, db.ErrSessionNotFound) {
			if !opts.CreateIfMissing {
				return nil, db.ErrSessionNotFound
			}

			newSess := &db.Session{
				AgentName:   opts.AgentName,
				Program:     opts.Program,
				Model:       opts.Model,
				ProjectPath: opts.ProjectPath,
			}
			if err := dbConn.CreateSession(newSess); err != nil {
				return nil, err
			}
			return newSess, nil
		}
		return nil, err
	}

	if opts.Program != "" && sess.Program != "" && sess.Program != opts.Program {
		if !opts.ForceEndMismatch {
			return nil, fmt.Errorf("%w: active=%q requested=%q", ErrSessionProgramMismatch, sess.Program, opts.Program)
		}

		// Force: end the old session and create a new one.
		if err := dbConn.EndSession(sess.ID); err != nil {
			return nil, err
		}
		newSess := &db.Session{
			AgentName:   opts.AgentName,
			Program:     opts.Program,
			Model:       opts.Model,
			ProjectPath: opts.ProjectPath,
		}
		if err := dbConn.CreateSession(newSess); err != nil {
			return nil, err
		}
		return newSess, nil
	}

	// Update heartbeat and return the refreshed session record.
	if err := dbConn.UpdateSessionHeartbeat(sess.ID); err != nil {
		return nil, err
	}
	return dbConn.GetSession(sess.ID)
}

// GarbageCollectStaleSessions finds stale sessions for a project and ends them unless DryRun is set.
func GarbageCollectStaleSessions(dbConn *db.DB, opts SessionGCOptions) (*SessionGCResult, error) {
	if dbConn == nil {
		return nil, fmt.Errorf("dbConn is required")
	}
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}
	if opts.Threshold <= 0 {
		return nil, fmt.Errorf("threshold must be > 0")
	}

	now := time.Now().UTC()
	res := &SessionGCResult{
		ProjectPath: opts.ProjectPath,
		Threshold:   opts.Threshold,
		Cutoff:      now.Add(-opts.Threshold),
	}

	stale, err := dbConn.FindStaleSessions(opts.Threshold)
	if err != nil {
		return nil, err
	}

	for _, s := range stale {
		if s.ProjectPath != opts.ProjectPath {
			continue
		}
		res.Sessions = append(res.Sessions, SessionSummary{
			ID:           s.ID,
			AgentName:    s.AgentName,
			Program:      s.Program,
			Model:        s.Model,
			ProjectPath:  s.ProjectPath,
			StartedAt:    s.StartedAt,
			LastActiveAt: s.LastActiveAt,
		})
	}

	if opts.DryRun || len(res.Sessions) == 0 {
		return res, nil
	}

	for _, s := range res.Sessions {
		if err := dbConn.EndSession(s.ID); err != nil {
			if errors.Is(err, db.ErrSessionNotFound) {
				res.SkippedIDs = append(res.SkippedIDs, s.ID)
				continue
			}
			return nil, err
		}
		res.EndedIDs = append(res.EndedIDs, s.ID)
	}

	return res, nil
}
