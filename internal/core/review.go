// Package core provides review submission and validation logic.
package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// Review errors.
var (
	ErrRequestNotPending  = errors.New("request is not pending")
	ErrSelfReview         = errors.New("cannot review your own request")
	ErrAlreadyReviewed    = errors.New("you have already reviewed this request")
	ErrRequireDiffModel   = errors.New("different model required for approval")
	ErrInvalidDecision    = errors.New("invalid decision (must be approve or reject)")
	ErrMissingSessionKey  = errors.New("session key required for signature")
)

// ConflictResolution specifies how to handle conflicting reviews.
type ConflictResolution string

const (
	// ConflictAnyRejectionBlocks means any rejection blocks approval (default).
	ConflictAnyRejectionBlocks ConflictResolution = "any_rejection_blocks"
	// ConflictFirstWins means the first response wins.
	ConflictFirstWins ConflictResolution = "first_wins"
	// ConflictHumanBreaksTie means escalate to human on conflict.
	ConflictHumanBreaksTie ConflictResolution = "human_breaks_tie"
)

// ReviewOptions contains parameters for submitting a review.
type ReviewOptions struct {
	// SessionID is the reviewer's session ID (required).
	SessionID string
	// SessionKey is the session's HMAC key for signing (required).
	SessionKey string
	// RequestID is the request being reviewed (required).
	RequestID string
	// Decision is approve or reject (required).
	Decision db.Decision
	// Responses contains structured responses to justification fields.
	Responses db.ReviewResponse
	// Comments contains optional additional comments.
	Comments string
}

// ReviewConfig provides configuration for the review process.
type ReviewConfig struct {
	// ConflictResolution specifies how to handle conflicting reviews.
	ConflictResolution ConflictResolution
	// TrustedSelfApprove lists agents that can self-approve after delay.
	TrustedSelfApprove []string
	// TrustedSelfApproveDelay is the delay before trusted agents can self-approve.
	TrustedSelfApproveDelay time.Duration
}

// DefaultReviewConfig returns the default review configuration.
func DefaultReviewConfig() ReviewConfig {
	return ReviewConfig{
		ConflictResolution:      ConflictAnyRejectionBlocks,
		TrustedSelfApprove:      nil,
		TrustedSelfApproveDelay: 5 * time.Minute,
	}
}

// ReviewResult contains the result of submitting a review.
type ReviewResult struct {
	// Review is the created review.
	Review *db.Review
	// RequestStatusChanged indicates if the request status was updated.
	RequestStatusChanged bool
	// NewRequestStatus is the new request status (if changed).
	NewRequestStatus db.RequestStatus
	// Approvals is the current approval count.
	Approvals int
	// Rejections is the current rejection count.
	Rejections int
}

// ReviewService handles review operations.
type ReviewService struct {
	db     *db.DB
	config ReviewConfig
}

// NewReviewService creates a new review service.
func NewReviewService(database *db.DB, config ReviewConfig) *ReviewService {
	return &ReviewService{
		db:     database,
		config: config,
	}
}

// SubmitReview validates and submits a review for a request.
// Returns the created review and any status change to the request.
func (rs *ReviewService) SubmitReview(opts ReviewOptions) (*ReviewResult, error) {
	// Validate required fields
	if opts.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if opts.RequestID == "" {
		return nil, errors.New("request_id is required")
	}
	if opts.SessionKey == "" {
		return nil, ErrMissingSessionKey
	}
	if opts.Decision != db.DecisionApprove && opts.Decision != db.DecisionReject {
		return nil, ErrInvalidDecision
	}

	// Step 1: Get and validate session
	session, err := rs.db.GetSession(opts.SessionID)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}
	if !session.IsActive() {
		return nil, ErrSessionInactive
	}

	// Step 2: Get and validate request
	request, err := rs.db.GetRequest(opts.RequestID)
	if err != nil {
		return nil, fmt.Errorf("getting request: %w", err)
	}
	if request.Status != db.StatusPending {
		return nil, fmt.Errorf("%w: status is %s", ErrRequestNotPending, request.Status)
	}

	// Step 3: Check not self-review (unless trusted self-approve agent)
	isSelfReview := opts.SessionID == request.RequestorSessionID
	if isSelfReview {
		if !rs.isTrustedSelfApprove(session.AgentName) {
			return nil, ErrSelfReview
		}
		// Trusted agents can self-approve after delay
		delay := rs.config.TrustedSelfApproveDelay
		if time.Since(request.CreatedAt) < delay {
			return nil, fmt.Errorf("trusted self-approve requires %v delay", delay)
		}
	}

	// Step 4: Check not already reviewed by this session
	alreadyReviewed, err := rs.db.HasReviewerAlreadyReviewed(opts.RequestID, opts.SessionID)
	if err != nil {
		return nil, fmt.Errorf("checking previous review: %w", err)
	}
	if alreadyReviewed {
		return nil, ErrAlreadyReviewed
	}

	// Step 5: Check require_different_model (for approvals only)
	if opts.Decision == db.DecisionApprove && request.RequireDifferentModel {
		if session.Model == request.RequestorModel {
			return nil, fmt.Errorf("%w: your model (%s) matches the requestor's", ErrRequireDiffModel, session.Model)
		}
	}

	// Step 6: Generate signature
	timestamp := time.Now().UTC()
	signature := db.ComputeReviewSignature(opts.SessionKey, opts.RequestID, opts.Decision, timestamp)

	// Step 7: Create review
	review := &db.Review{
		RequestID:          opts.RequestID,
		ReviewerSessionID:  opts.SessionID,
		ReviewerAgent:      session.AgentName,
		ReviewerModel:      session.Model,
		Decision:           opts.Decision,
		Signature:          signature,
		SignatureTimestamp: timestamp,
		Responses:          opts.Responses,
		Comments:           opts.Comments,
	}

	if err := rs.db.CreateReview(review); err != nil {
		return nil, fmt.Errorf("creating review: %w", err)
	}

	// Step 8: Check if decision changes request state
	result := &ReviewResult{
		Review: review,
	}

	approvals, rejections, err := rs.db.CountReviewsByDecision(opts.RequestID)
	if err != nil {
		return nil, fmt.Errorf("counting reviews: %w", err)
	}
	result.Approvals = approvals
	result.Rejections = rejections

	// Apply conflict resolution rules
	newStatus := rs.determineNewStatus(request, opts.Decision, approvals, rejections)
	if newStatus != "" && newStatus != request.Status {
		if err := rs.db.UpdateRequestStatus(opts.RequestID, newStatus); err != nil {
			return nil, fmt.Errorf("updating request status: %w", err)
		}
		result.RequestStatusChanged = true
		result.NewRequestStatus = newStatus
	}

	return result, nil
}

// isTrustedSelfApprove checks if an agent is in the trusted self-approve list.
func (rs *ReviewService) isTrustedSelfApprove(agentName string) bool {
	for _, trusted := range rs.config.TrustedSelfApprove {
		if trusted == agentName {
			return true
		}
	}
	return false
}

// determineNewStatus determines what status the request should transition to.
func (rs *ReviewService) determineNewStatus(
	request *db.Request,
	decision db.Decision,
	approvals, rejections int,
) db.RequestStatus {
	switch rs.config.ConflictResolution {
	case ConflictAnyRejectionBlocks:
		// Any rejection immediately blocks
		if rejections > 0 {
			return db.StatusRejected
		}
		// Check if we have enough approvals
		if approvals >= request.MinApprovals {
			return db.StatusApproved
		}

	case ConflictFirstWins:
		// First review determines outcome
		if approvals+rejections == 1 {
			if decision == db.DecisionApprove {
				return db.StatusApproved
			}
			return db.StatusRejected
		}

	case ConflictHumanBreaksTie:
		// If there's a mix of approvals and rejections, escalate
		if approvals > 0 && rejections > 0 {
			return db.StatusEscalated
		}
		// Otherwise, check if we have enough approvals
		if approvals >= request.MinApprovals {
			return db.StatusApproved
		}
		// Or if any rejections
		if rejections > 0 {
			return db.StatusRejected
		}
	}

	return "" // No status change
}

// VerifyReview validates a review's signature.
func VerifyReview(review *db.Review, sessionKey string) bool {
	return db.VerifyReviewSignature(
		sessionKey,
		review.RequestID,
		review.Decision,
		review.SignatureTimestamp,
		review.Signature,
	)
}

// CanReview checks if a session can submit a review for a request.
func (rs *ReviewService) CanReview(sessionID, requestID string) (bool, string) {
	// Get session
	session, err := rs.db.GetSession(sessionID)
	if err != nil {
		return false, fmt.Sprintf("session not found: %v", err)
	}
	if !session.IsActive() {
		return false, "session is not active"
	}

	// Get request
	request, err := rs.db.GetRequest(requestID)
	if err != nil {
		return false, fmt.Sprintf("request not found: %v", err)
	}
	if request.Status != db.StatusPending {
		return false, fmt.Sprintf("request is not pending (status: %s)", request.Status)
	}

	// Check self-review
	if sessionID == request.RequestorSessionID {
		if !rs.isTrustedSelfApprove(session.AgentName) {
			return false, "cannot review your own request"
		}
		delay := rs.config.TrustedSelfApproveDelay
		if time.Since(request.CreatedAt) < delay {
			return false, fmt.Sprintf("trusted self-approve requires %v delay", delay)
		}
	}

	// Check already reviewed
	alreadyReviewed, err := rs.db.HasReviewerAlreadyReviewed(requestID, sessionID)
	if err != nil {
		return false, fmt.Sprintf("error checking previous review: %v", err)
	}
	if alreadyReviewed {
		return false, "you have already reviewed this request"
	}

	return true, ""
}

// GetReviewStatus returns the current review status for a request.
type ReviewStatus struct {
	// RequestStatus is the current request status.
	RequestStatus db.RequestStatus
	// Approvals is the current approval count.
	Approvals int
	// Rejections is the current rejection count.
	Rejections int
	// MinApprovals is the required approval count.
	MinApprovals int
	// NeedsMoreApprovals indicates if more approvals are needed.
	NeedsMoreApprovals bool
	// Reviews contains all reviews for the request.
	Reviews []*db.Review
}

// GetReviewStatus retrieves the current review status for a request.
func (rs *ReviewService) GetReviewStatus(requestID string) (*ReviewStatus, error) {
	request, err := rs.db.GetRequest(requestID)
	if err != nil {
		return nil, fmt.Errorf("getting request: %w", err)
	}

	reviews, err := rs.db.ListReviewsForRequest(requestID)
	if err != nil {
		return nil, fmt.Errorf("listing reviews: %w", err)
	}

	approvals, rejections, err := rs.db.CountReviewsByDecision(requestID)
	if err != nil {
		return nil, fmt.Errorf("counting reviews: %w", err)
	}

	return &ReviewStatus{
		RequestStatus:      request.Status,
		Approvals:          approvals,
		Rejections:         rejections,
		MinApprovals:       request.MinApprovals,
		NeedsMoreApprovals: approvals < request.MinApprovals && request.Status == db.StatusPending,
		Reviews:            reviews,
	}, nil
}
