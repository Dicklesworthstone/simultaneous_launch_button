// Package core implements the core domain logic for SLB.
package core

import (
	"github.com/Dicklesworthstone/slb/internal/db"
)

// Re-export db types for convenience.
// This allows callers to use core.RiskTier instead of db.RiskTier.
type (
	// RiskTier represents the risk classification of a command.
	RiskTier = db.RiskTier
	// RequestStatus represents the current state of a request.
	RequestStatus = db.RequestStatus
	// Decision represents an approval or rejection decision.
	Decision = db.Decision
	// Session represents an agent session.
	Session = db.Session
	// Request represents a command request.
	Request = db.Request
	// Review represents an approval or rejection.
	Review = db.Review
	// CommandSpec represents the command specification.
	CommandSpec = db.CommandSpec
	// Justification represents the reasoning for a request.
	Justification = db.Justification
)

// Re-export constants for convenience.
const (
	// Risk tiers
	RiskTierCritical  = db.RiskTierCritical
	RiskTierDangerous = db.RiskTierDangerous
	RiskTierCaution   = db.RiskTierCaution

	// Request statuses
	StatusPending         = db.StatusPending
	StatusApproved        = db.StatusApproved
	StatusRejected        = db.StatusRejected
	StatusExecuting       = db.StatusExecuting
	StatusExecuted        = db.StatusExecuted
	StatusExecutionFailed = db.StatusExecutionFailed
	StatusCancelled       = db.StatusCancelled
	StatusTimeout         = db.StatusTimeout
	StatusTimedOut        = db.StatusTimedOut
	StatusEscalated       = db.StatusEscalated

	// Decisions
	DecisionApprove = db.DecisionApprove
	DecisionReject  = db.DecisionReject
)

// RiskSafe represents commands that are safe and skip approval entirely.
// This is separate from the three database tiers.
const RiskSafe = "safe"

// ClassifyRisk determines the risk tier for a command.
// This is a placeholder that will be implemented in the pattern matching engine.
func ClassifyRisk(command string) RiskTier {
	// Default to dangerous - pattern engine will override
	return RiskTierDangerous
}

// MinApprovalsForTier returns the minimum approvals required for a risk tier.
func MinApprovalsForTier(tier RiskTier) int {
	if tier == RiskTier(RiskSafe) {
		return 0
	}
	return tier.MinApprovals()
}

// IsSafeTier reports whether the tier represents a safe/no-approval command.
func IsSafeTier(tier RiskTier) bool {
	return tier == RiskTier(RiskSafe)
}
