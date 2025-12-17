// Package e2e provides end-to-end testing infrastructure for SLB.
//
// The E2E tests validate full workflows from request to execution, covering:
//   - Multi-agent approval flows
//   - Risk tier classification behavior
//   - Session timeout handling
//   - Git rollback functionality
//
// # Test Structure
//
// Tests are organized as:
//
//	tests/e2e/
//	├── harness/           # Test infrastructure
//	│   ├── harness.go     # Environment setup
//	│   ├── logging.go     # Step logging
//	│   └── assertions.go  # Domain assertions
//	└── scenarios/         # Test scenarios
//
// # Usage
//
// Each E2E test should create a new environment:
//
//	func TestMultiAgentApproval(t *testing.T) {
//	    env := harness.NewE2EEnvironment(t)
//	    defer env.Cleanup()
//
//	    // Test steps...
//	    env.Step("Creating requestor session")
//	    sess := env.CreateSession("AgentA", "claude-code", "opus")
//
//	    env.Step("Submitting request")
//	    req := env.SubmitRequest(sess, "rm -rf ./build", "Cleanup build artifacts")
//
//	    env.AssertRequestTier(req, db.RiskTierDangerous)
//	}
//
// # Design Principles
//
//   - Isolation: Each test gets its own temp directory, DB, and git repo
//   - Cleanup: All resources are cleaned up via t.Cleanup
//   - Logging: Every step is logged with timestamps for debugging
//   - Timeouts: Short timeouts (5s max) to catch hangs
//   - Determinism: No random behavior that could cause flakiness
package e2e
