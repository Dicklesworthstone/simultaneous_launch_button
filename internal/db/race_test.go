package db

import (
	"errors"
	"sync"
	"testing"
)

func TestUpdateRequestStatus_RaceCondition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Transition to Approved
	if err := db.UpdateRequestStatus(r.ID, StatusApproved); err != nil {
		t.Fatalf("Failed to approve request: %v", err)
	}

	// Try to execute from two goroutines concurrently
	var wg sync.WaitGroup
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- db.UpdateRequestStatus(r.ID, StatusExecuting)
		}()
	}

	wg.Wait()
	close(results)

	successCount := 0
	errorCount := 0
	var lastErr error

	for err := range results {
		if err == nil {
			successCount++
		} else {
			errorCount++
			lastErr = err
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success, got %d", successCount)
	}
	if errorCount != 1 {
		t.Errorf("Expected exactly 1 error, got %d", errorCount)
	}

	// Verify the error is related to invalid transition (concurrent update)
	if !errors.Is(lastErr, ErrInvalidTransition) {
		t.Errorf("Expected ErrInvalidTransition, got %v", lastErr)
	}
}
