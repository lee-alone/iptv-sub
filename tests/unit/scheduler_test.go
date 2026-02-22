package unit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestAddJob_NewJob tests adding a new job to the scheduler
func TestAddJob_NewJob(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error {
		return nil
	}

	err := scheduler.AddJob("test-job", "1s", fn)
	tests.AssertNoError(t, err, "AddJob should not return error for new job")
}

// TestAddJob_DuplicateJob tests that adding a duplicate job returns an error
func TestAddJob_DuplicateJob(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }

	err1 := scheduler.AddJob("test-job", "1s", fn)
	tests.AssertNoError(t, err1, "First AddJob should succeed")

	err2 := scheduler.AddJob("test-job", "1s", fn)
	tests.AssertError(t, err2, "Second AddJob with same ID should return error")
	tests.AssertStringContains(t, err2.Error(), "already exists", "Error should mention job already exists")
}

// TestAddJob_InvalidSchedule tests that invalid schedule format returns an error
func TestAddJob_InvalidSchedule(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }

	err := scheduler.AddJob("test-job", "invalid", fn)
	tests.AssertError(t, err, "AddJob with invalid schedule should return error")
}

// TestAddJob_NegativeDuration tests that negative duration returns an error
func TestAddJob_NegativeDuration(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }

	err := scheduler.AddJob("test-job", "-1s", fn)
	tests.AssertError(t, err, "AddJob with negative duration should return error")
}

// TestRemoveJob_ExistingJob tests removing an existing job
func TestRemoveJob_ExistingJob(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }
	scheduler.AddJob("test-job", "1s", fn)

	err := scheduler.RemoveJob("test-job")
	tests.AssertNoError(t, err, "RemoveJob should not return error for existing job")
}

// TestRemoveJob_NonExistentJob tests that removing a non-existent job returns an error
func TestRemoveJob_NonExistentJob(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	err := scheduler.RemoveJob("non-existent")
	tests.AssertError(t, err, "RemoveJob should return error for non-existent job")
	tests.AssertStringContains(t, err.Error(), "not found", "Error should mention job not found")
}

// TestRemoveJob_AfterRemoval tests that a removed job cannot be removed again
func TestRemoveJob_AfterRemoval(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }
	scheduler.AddJob("test-job", "1s", fn)

	err1 := scheduler.RemoveJob("test-job")
	tests.AssertNoError(t, err1, "First RemoveJob should succeed")

	err2 := scheduler.RemoveJob("test-job")
	tests.AssertError(t, err2, "Second RemoveJob should return error")
}

// TestTriggerJob_ManualExecution tests manually triggering a job
func TestTriggerJob_ManualExecution(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	executed := false
	fn := func() error {
		executed = true
		return nil
	}

	scheduler.AddJob("test-job", "1h", fn)
	time.Sleep(100 * time.Millisecond) // Allow job to be added

	err := scheduler.TriggerJob("test-job")
	tests.AssertNoError(t, err, "TriggerJob should not return error")

	time.Sleep(100 * time.Millisecond) // Allow async execution
	tests.AssertTrue(t, executed, "Job should be executed after trigger")
}

// TestTriggerJob_NonExistentJob tests triggering a non-existent job
func TestTriggerJob_NonExistentJob(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	err := scheduler.TriggerJob("non-existent")
	tests.AssertError(t, err, "TriggerJob should return error for non-existent job")
}

// TestScheduledExecution_TimedExecution tests that jobs execute at scheduled times
func TestScheduledExecution_TimedExecution(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	executionCount := int32(0)
	fn := func() error {
		atomic.AddInt32(&executionCount, 1)
		return nil
	}

	scheduler.AddJob("test-job", "100ms", fn)
	scheduler.Start()

	time.Sleep(350 * time.Millisecond)

	count := atomic.LoadInt32(&executionCount)
	// Should execute at least 3 times (initial + 2 scheduled)
	tests.AssertTrue(t, count >= 3, fmt.Sprintf("Job should execute at least 3 times, got %d", count))
}

// TestScheduledExecution_MultipleJobs tests scheduling multiple jobs
func TestScheduledExecution_MultipleJobs(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	count1 := int32(0)
	count2 := int32(0)

	fn1 := func() error {
		atomic.AddInt32(&count1, 1)
		return nil
	}

	fn2 := func() error {
		atomic.AddInt32(&count2, 1)
		return nil
	}

	scheduler.AddJob("job1", "100ms", fn1)
	scheduler.AddJob("job2", "100ms", fn2)
	scheduler.Start()

	time.Sleep(250 * time.Millisecond)

	c1 := atomic.LoadInt32(&count1)
	c2 := atomic.LoadInt32(&count2)

	tests.AssertTrue(t, c1 > 0, "Job 1 should execute")
	tests.AssertTrue(t, c2 > 0, "Job 2 should execute")
}

// TestJobExecution_WithError tests job execution when function returns error
func TestJobExecution_WithError(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	executionCount := int32(0)
	fn := func() error {
		atomic.AddInt32(&executionCount, 1)
		return fmt.Errorf("test error")
	}

	scheduler.AddJob("test-job", "100ms", fn)
	scheduler.Start()

	time.Sleep(250 * time.Millisecond)

	count := atomic.LoadInt32(&executionCount)
	// Job should continue executing even if it returns error
	tests.AssertTrue(t, count >= 2, fmt.Sprintf("Job should execute multiple times despite error, got %d", count))
}

// TestScheduler_Start tests starting the scheduler
func TestScheduler_Start(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }
	scheduler.AddJob("test-job", "1s", fn)

	err := scheduler.Start()
	tests.AssertNoError(t, err, "Start should not return error")
}

// TestScheduler_Stop tests stopping the scheduler
func TestScheduler_Stop(t *testing.T) {
	scheduler := services.NewScheduler()

	fn := func() error { return nil }
	scheduler.AddJob("test-job", "100ms", fn)
	scheduler.Start()

	time.Sleep(150 * time.Millisecond)

	err := scheduler.Stop()
	tests.AssertNoError(t, err, "Stop should not return error")
}

// TestScheduler_ConcurrentOperations tests concurrent add/remove operations
func TestScheduler_ConcurrentOperations(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	var wg sync.WaitGroup
	errors := make([]error, 0)
	mu := sync.Mutex{}

	fn := func() error { return nil }

	// Add jobs concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			jobID := fmt.Sprintf("job-%d", id)
			err := scheduler.AddJob(jobID, "1s", fn)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	tests.AssertEqual(t, 0, len(errors), "No errors should occur during concurrent adds")
}

// TestAddJob_MultipleValidSchedules tests adding jobs with various valid schedule formats
func TestAddJob_MultipleValidSchedules(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }

	schedules := []string{"1s", "10s", "1m", "1h"}
	for i, schedule := range schedules {
		jobID := fmt.Sprintf("job-%d", i)
		err := scheduler.AddJob(jobID, schedule, fn)
		tests.AssertNoError(t, err, fmt.Sprintf("AddJob should succeed with schedule %s", schedule))
	}
}

// TestRemoveJob_MultipleJobs tests removing multiple jobs
func TestRemoveJob_MultipleJobs(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	fn := func() error { return nil }

	// Add multiple jobs
	for i := 0; i < 5; i++ {
		jobID := fmt.Sprintf("job-%d", i)
		scheduler.AddJob(jobID, "1s", fn)
	}

	// Remove all jobs
	for i := 0; i < 5; i++ {
		jobID := fmt.Sprintf("job-%d", i)
		err := scheduler.RemoveJob(jobID)
		tests.AssertNoError(t, err, fmt.Sprintf("RemoveJob should succeed for job %s", jobID))
	}
}

// TestTriggerJob_MultipleExecutions tests triggering the same job multiple times
func TestTriggerJob_MultipleExecutions(t *testing.T) {
	scheduler := services.NewScheduler()
	defer scheduler.Stop()

	executionCount := int32(0)
	fn := func() error {
		atomic.AddInt32(&executionCount, 1)
		return nil
	}

	scheduler.AddJob("test-job", "1h", fn)

	// Trigger multiple times
	for i := 0; i < 5; i++ {
		err := scheduler.TriggerJob("test-job")
		tests.AssertNoError(t, err, "TriggerJob should succeed")
	}

	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&executionCount)
	tests.AssertTrue(t, count >= 5, fmt.Sprintf("Job should execute at least 5 times, got %d", count))
}
