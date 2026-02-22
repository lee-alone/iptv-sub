package unit

import (
	"fmt"
	"sync"
	"testing"

	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestAddSubscription_NewSubscription tests adding a new subscription
func TestAddSubscription_NewSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	err := sm.AddSubscription("http://example.com/playlist.m3u", "Example", true)
	tests.AssertNoError(t, err, "AddSubscription should succeed for new subscription")
}

// TestAddSubscription_DuplicateSubscription tests that adding duplicate subscription returns error
func TestAddSubscription_DuplicateSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	err := sm.AddSubscription(url, "Example", true)
	tests.AssertError(t, err, "AddSubscription should return error for duplicate")
	tests.AssertStringContains(t, err.Error(), "already exists", "Error should mention already exists")
}

// TestGetSubscription_ExistingSubscription tests retrieving an existing subscription
func TestGetSubscription_ExistingSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	sub := sm.GetSubscription(url)
	tests.AssertNotNil(t, sub, "GetSubscription should return subscription")
	tests.AssertEqual(t, url, sub.URL, "Subscription URL should match")
	tests.AssertEqual(t, "Example", sub.Name, "Subscription name should match")
}

// TestGetSubscription_NonExistentSubscription tests retrieving non-existent subscription
func TestGetSubscription_NonExistentSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	sub := sm.GetSubscription("http://nonexistent.com/playlist.m3u")
	if sub != nil {
		t.Errorf("GetSubscription should return nil for non-existent subscription, got %v", sub)
	}
}

// TestGetAllSubscriptions_EmptyList tests getting all subscriptions when empty
func TestGetAllSubscriptions_EmptyList(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 0, len(subs), "GetAllSubscriptions should return empty list")
}

// TestGetAllSubscriptions_MultipleSubscriptions tests getting all subscriptions
func TestGetAllSubscriptions_MultipleSubscriptions(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		sm.AddSubscription(url, fmt.Sprintf("Example %d", i), true)
	}

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 5, len(subs), "GetAllSubscriptions should return all subscriptions")
}

// TestUpdateSubscription_ChangeURL tests updating subscription URL
func TestUpdateSubscription_ChangeURL(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	oldURL := "http://example.com/old.m3u"
	newURL := "http://example.com/new.m3u"

	sm.AddSubscription(oldURL, "Example", true)
	err := sm.UpdateSubscription(oldURL, newURL, "Example Updated", true)

	tests.AssertNoError(t, err, "UpdateSubscription should succeed")

	oldSub := sm.GetSubscription(oldURL)
	newSub := sm.GetSubscription(newURL)

	if oldSub != nil {
		t.Errorf("Old subscription should not exist, got %v", oldSub)
	}
	tests.AssertNotNil(t, newSub, "New subscription should exist")
	tests.AssertEqual(t, newURL, newSub.URL, "URL should be updated")
}

// TestUpdateSubscription_ChangeName tests updating subscription name
func TestUpdateSubscription_ChangeName(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Old Name", true)

	err := sm.UpdateSubscription(url, url, "New Name", true)
	tests.AssertNoError(t, err, "UpdateSubscription should succeed")

	sub := sm.GetSubscription(url)
	tests.AssertEqual(t, "New Name", sub.Name, "Name should be updated")
}

// TestUpdateSubscription_ChangeEnabled tests updating subscription enabled status
func TestUpdateSubscription_ChangeEnabled(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	err := sm.UpdateSubscription(url, url, "Example", false)
	tests.AssertNoError(t, err, "UpdateSubscription should succeed")

	sub := sm.GetSubscription(url)
	tests.AssertFalse(t, sub.Enabled, "Enabled should be false")
}

// TestUpdateSubscription_NonExistentSubscription tests updating non-existent subscription
func TestUpdateSubscription_NonExistentSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	err := sm.UpdateSubscription("http://nonexistent.com/playlist.m3u", "http://new.com/playlist.m3u", "New", true)
	tests.AssertError(t, err, "UpdateSubscription should return error for non-existent subscription")
}

// TestUpdateSubscription_DuplicateNewURL tests updating to URL that already exists
func TestUpdateSubscription_DuplicateNewURL(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url1 := "http://example1.com/playlist.m3u"
	url2 := "http://example2.com/playlist.m3u"

	sm.AddSubscription(url1, "Example 1", true)
	sm.AddSubscription(url2, "Example 2", true)

	err := sm.UpdateSubscription(url1, url2, "Example 1", true)
	tests.AssertError(t, err, "UpdateSubscription should return error when new URL already exists")
}

// TestRemoveSubscription_ExistingSubscription tests removing an existing subscription
func TestRemoveSubscription_ExistingSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	err := sm.RemoveSubscription(url)
	tests.AssertNoError(t, err, "RemoveSubscription should succeed")

	sub := sm.GetSubscription(url)
	if sub != nil {
		t.Errorf("Subscription should be removed, got %v", sub)
	}
}

// TestRemoveSubscription_NonExistentSubscription tests removing non-existent subscription
func TestRemoveSubscription_NonExistentSubscription(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	err := sm.RemoveSubscription("http://nonexistent.com/playlist.m3u")
	tests.AssertError(t, err, "RemoveSubscription should return error for non-existent subscription")
}

// TestUpdateSubscriptionStatus_Success tests updating subscription status to success
func TestUpdateSubscriptionStatus_Success(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	err := sm.UpdateSubscriptionStatus(url, "success", 100)
	tests.AssertNoError(t, err, "UpdateSubscriptionStatus should succeed")

	sub := sm.GetSubscription(url)
	tests.AssertEqual(t, "success", sub.Status, "Status should be updated")
	tests.AssertEqual(t, 100, sub.ChannelCount, "Channel count should be updated")
}

// TestUpdateSubscriptionStatus_Failed tests updating subscription status to failed
func TestUpdateSubscriptionStatus_Failed(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	err := sm.UpdateSubscriptionStatus(url, "failed", 0)
	tests.AssertNoError(t, err, "UpdateSubscriptionStatus should succeed")

	sub := sm.GetSubscription(url)
	tests.AssertEqual(t, "failed", sub.Status, "Status should be updated to failed")
}

// TestLoad_PersistenceAfterReload tests that subscriptions persist after reload
func TestLoad_PersistenceAfterReload(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create and add subscriptions
	sm1 := services.NewSubscriptionManager(tmpDir)
	url := "http://example.com/playlist.m3u"
	sm1.AddSubscription(url, "Example", true)

	// Create new manager instance (should load from file)
	sm2 := services.NewSubscriptionManager(tmpDir)
	sub := sm2.GetSubscription(url)

	tests.AssertNotNil(t, sub, "Subscription should be loaded from file")
	tests.AssertEqual(t, url, sub.URL, "URL should match")
	tests.AssertEqual(t, "Example", sub.Name, "Name should match")
}

// TestConcurrentOperations_ReadWrite tests concurrent read/write operations
func TestConcurrentOperations_ReadWrite(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	var wg sync.WaitGroup
	errors := make([]error, 0)
	mu := sync.Mutex{}

	// Add subscriptions concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			url := fmt.Sprintf("http://example%d.com/playlist.m3u", id)
			err := sm.AddSubscription(url, fmt.Sprintf("Example %d", id), true)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	tests.AssertEqual(t, 0, len(errors), "No errors should occur during concurrent adds")

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 10, len(subs), "All subscriptions should be added")
}

// TestConcurrentOperations_DataConsistency tests data consistency with concurrent operations
func TestConcurrentOperations_DataConsistency(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	// Add initial subscriptions
	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		sm.AddSubscription(url, fmt.Sprintf("Example %d", i), true)
	}

	var wg sync.WaitGroup
	errors := make([]error, 0)
	mu := sync.Mutex{}

	// Concurrent reads and writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			url := fmt.Sprintf("http://example%d.com/playlist.m3u", id)
			sub := sm.GetSubscription(url)
			if sub == nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("subscription not found: %s", url))
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	tests.AssertEqual(t, 0, len(errors), "No errors should occur during concurrent reads")
}

// TestCRUD_CompleteWorkflow tests complete CRUD workflow
func TestCRUD_CompleteWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	// Create
	url := "http://example.com/playlist.m3u"
	err := sm.AddSubscription(url, "Example", true)
	tests.AssertNoError(t, err, "Create should succeed")

	// Read
	sub := sm.GetSubscription(url)
	tests.AssertNotNil(t, sub, "Read should return subscription")

	// Update
	err = sm.UpdateSubscription(url, url, "Updated Example", false)
	tests.AssertNoError(t, err, "Update should succeed")

	sub = sm.GetSubscription(url)
	tests.AssertEqual(t, "Updated Example", sub.Name, "Name should be updated")
	tests.AssertFalse(t, sub.Enabled, "Enabled should be false")

	// Delete
	err = sm.RemoveSubscription(url)
	tests.AssertNoError(t, err, "Delete should succeed")

	sub = sm.GetSubscription(url)
	if sub != nil {
		t.Errorf("Subscription should be deleted, got %v", sub)
	}
}

// TestAddSubscription_MultipleSubscriptions tests adding multiple subscriptions
func TestAddSubscription_MultipleSubscriptions(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	for i := 0; i < 10; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		err := sm.AddSubscription(url, fmt.Sprintf("Example %d", i), true)
		tests.AssertNoError(t, err, fmt.Sprintf("AddSubscription should succeed for subscription %d", i))
	}

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 10, len(subs), "All subscriptions should be added")
}

// TestRemoveSubscription_MultipleSubscriptions tests removing multiple subscriptions
func TestRemoveSubscription_MultipleSubscriptions(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	// Add subscriptions
	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		sm.AddSubscription(url, fmt.Sprintf("Example %d", i), true)
	}

	// Remove all
	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		err := sm.RemoveSubscription(url)
		tests.AssertNoError(t, err, fmt.Sprintf("RemoveSubscription should succeed for subscription %d", i))
	}

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 0, len(subs), "All subscriptions should be removed")
}

// TestUpdateSubscriptionStatus_MultipleUpdates tests multiple status updates
func TestUpdateSubscriptionStatus_MultipleUpdates(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"
	sm.AddSubscription(url, "Example", true)

	// Update status multiple times
	for i := 0; i < 5; i++ {
		err := sm.UpdateSubscriptionStatus(url, "success", i*10)
		tests.AssertNoError(t, err, "UpdateSubscriptionStatus should succeed")

		sub := sm.GetSubscription(url)
		tests.AssertEqual(t, i*10, sub.ChannelCount, fmt.Sprintf("Channel count should be %d", i*10))
	}
}

// TestLoad_EmptyFile tests loading when no subscriptions file exists
func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	sm := services.NewSubscriptionManager(tmpDir)

	subs := sm.GetAllSubscriptions()
	tests.AssertEqual(t, 0, len(subs), "Should return empty list when no file exists")
}

// TestPersistence_AfterMultipleOperations tests persistence after multiple operations
func TestPersistence_AfterMultipleOperations(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// First manager instance
	sm1 := services.NewSubscriptionManager(tmpDir)
	for i := 0; i < 3; i++ {
		url := fmt.Sprintf("http://example%d.com/playlist.m3u", i)
		sm1.AddSubscription(url, fmt.Sprintf("Example %d", i), true)
	}

	// Second manager instance (should load from file)
	sm2 := services.NewSubscriptionManager(tmpDir)
	subs := sm2.GetAllSubscriptions()
	tests.AssertEqual(t, 3, len(subs), "Should load all subscriptions from file")

	// Add more subscriptions
	sm2.AddSubscription("http://example3.com/playlist.m3u", "Example 3", true)

	// Third manager instance (should load all)
	sm3 := services.NewSubscriptionManager(tmpDir)
	subs = sm3.GetAllSubscriptions()
	tests.AssertEqual(t, 4, len(subs), "Should load all subscriptions including new ones")
}
