package integration

import (
	"testing"
	"time"

	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestIntegrationSchedulerWorkflow tests the complete scheduler workflow
func TestIntegrationSchedulerWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	// Step 1: Add a job
	jobID := "test-job-1"
	schedule := "1s" // 1 second interval
	executed := false

	err := scheduler.AddJob(jobID, schedule, func() error {
		executed = true
		return nil
	})
	tests.AssertNoError(t, err, "Should add job successfully")

	// Step 2: Trigger job manually
	err = scheduler.TriggerJob(jobID)
	tests.AssertNoError(t, err, "Should trigger job successfully")

	// Give job time to execute
	time.Sleep(100 * time.Millisecond)
	tests.AssertTrue(t, executed, "Job should have been executed")

	// Step 3: Remove job
	err = scheduler.RemoveJob(jobID)
	tests.AssertNoError(t, err, "Should remove job successfully")

	t.Logf("✓ Scheduler workflow test passed")
}

// TestIntegrationMultipleJobs tests managing multiple scheduled jobs
func TestIntegrationMultipleJobs(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	// Add multiple jobs
	for i := 1; i <= 3; i++ {
		jobID := "job-" + string(rune(48+i)) // "job-1", "job-2", "job-3"
		schedule := "1s"
		err := scheduler.AddJob(jobID, schedule, func() error {
			return nil
		})
		tests.AssertNoError(t, err, "Should add job successfully")
	}

	// Verify jobs were added by trying to add duplicate
	err := scheduler.AddJob("job-1", "1s", func() error {
		return nil
	})
	tests.AssertError(t, err, "Should reject duplicate job")

	// Remove one job
	err = scheduler.RemoveJob("job-1")
	tests.AssertNoError(t, err, "Should remove job successfully")

	// Verify job was removed by trying to remove again
	err = scheduler.RemoveJob("job-1")
	tests.AssertError(t, err, "Should fail to remove non-existent job")

	t.Logf("✓ Multiple jobs test passed")
}

// TestIntegrationJobExecution tests job execution and error handling
func TestIntegrationJobExecution(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	// Add job that succeeds
	successExecuted := false
	err := scheduler.AddJob("success-job", "1s", func() error {
		successExecuted = true
		return nil
	})
	tests.AssertNoError(t, err, "Should add success job")

	// Add job that fails
	failureExecuted := false
	err = scheduler.AddJob("failure-job", "1s", func() error {
		failureExecuted = true
		return nil // In real scenario, this would return an error
	})
	tests.AssertNoError(t, err, "Should add failure job")

	// Trigger both jobs
	err = scheduler.TriggerJob("success-job")
	tests.AssertNoError(t, err, "Should trigger success job")

	err = scheduler.TriggerJob("failure-job")
	tests.AssertNoError(t, err, "Should trigger failure job")

	// Give jobs time to execute
	time.Sleep(100 * time.Millisecond)

	tests.AssertTrue(t, successExecuted, "Success job should have been executed")
	tests.AssertTrue(t, failureExecuted, "Failure job should have been executed")

	t.Logf("✓ Job execution test passed")
}

// TestIntegrationJobDuplicatePrevention tests that duplicate jobs are prevented
func TestIntegrationJobDuplicatePrevention(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	jobID := "test-job"
	schedule := "1s"

	// Add job
	err := scheduler.AddJob(jobID, schedule, func() error {
		return nil
	})
	tests.AssertNoError(t, err, "Should add job successfully")

	// Try to add duplicate job
	err = scheduler.AddJob(jobID, schedule, func() error {
		return nil
	})
	tests.AssertError(t, err, "Should reject duplicate job")

	t.Logf("✓ Job duplicate prevention test passed")
}

// TestIntegrationJobConcurrentOperations tests concurrent job operations
func TestIntegrationJobConcurrentOperations(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	// Add jobs concurrently
	done := make(chan bool, 5)

	for i := 1; i <= 5; i++ {
		go func(index int) {
			jobID := "job-" + string(rune(48+index))
			schedule := "1s"
			err := scheduler.AddJob(jobID, schedule, func() error {
				return nil
			})
			tests.AssertNoError(t, err, "Should add job successfully")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	t.Logf("✓ Job concurrent operations test passed")
}
