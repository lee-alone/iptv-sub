package integration

import (
	"testing"

	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestIntegrationSubscriptionManagementWorkflow tests the complete subscription management workflow
func TestIntegrationSubscriptionManagementWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	subscriptionMgr := services.NewSubscriptionManager(tmpDir)

	// Step 1: Add subscription
	url1 := "http://example.com/playlist1.m3u"
	err := subscriptionMgr.AddSubscription(url1, "Subscription 1", true)
	tests.AssertNoError(t, err, "Should add subscription successfully")

	// Verify subscription was added
	subs := subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have 1 subscription")
	tests.AssertEqual(t, url1, subs[0].URL, "Subscription URL should match")
	tests.AssertEqual(t, "Subscription 1", subs[0].Name, "Subscription name should match")
	tests.AssertTrue(t, subs[0].Enabled, "Subscription should be enabled")

	// Step 2: Add another subscription
	url2 := "http://example.com/playlist2.m3u"
	err = subscriptionMgr.AddSubscription(url2, "Subscription 2", false)
	tests.AssertNoError(t, err, "Should add second subscription successfully")

	// Verify both subscriptions exist
	subs = subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 2, len(subs), "Should have 2 subscriptions")

	// Step 3: Get specific subscription
	sub := subscriptionMgr.GetSubscription(url1)
	tests.AssertNotNil(t, sub, "Should retrieve subscription")
	tests.AssertEqual(t, "Subscription 1", sub.Name, "Retrieved subscription name should match")

	// Step 4: Update subscription
	err = subscriptionMgr.UpdateSubscription(url1, url1, "Updated Subscription 1", false)
	tests.AssertNoError(t, err, "Should update subscription successfully")

	// Verify subscription was updated
	sub = subscriptionMgr.GetSubscription(url1)
	tests.AssertNotNil(t, sub, "Should retrieve updated subscription")
	tests.AssertEqual(t, "Updated Subscription 1", sub.Name, "Updated subscription name should match")
	tests.AssertFalse(t, sub.Enabled, "Updated subscription should be disabled")

	// Step 5: Remove subscription
	err = subscriptionMgr.RemoveSubscription(url1)
	tests.AssertNoError(t, err, "Should remove subscription successfully")

	// Verify subscription was removed
	subs = subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have 1 subscription after removal")
	tests.AssertEqual(t, url2, subs[0].URL, "Remaining subscription should be the second one")

	// Step 6: Remove all subscriptions
	err = subscriptionMgr.RemoveSubscription(url2)
	tests.AssertNoError(t, err, "Should remove second subscription successfully")

	subs = subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 0, len(subs), "Should have 0 subscriptions after removing all")

	t.Logf("✓ Subscription management workflow test passed")
}

// TestIntegrationSubscriptionPersistence tests that subscriptions are persisted to disk
func TestIntegrationSubscriptionPersistence(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create first manager and add subscriptions
	mgr1 := services.NewSubscriptionManager(tmpDir)
	url1 := "http://example.com/playlist1.m3u"
	url2 := "http://example.com/playlist2.m3u"

	err := mgr1.AddSubscription(url1, "Subscription 1", true)
	tests.AssertNoError(t, err, "Should add first subscription")

	err = mgr1.AddSubscription(url2, "Subscription 2", false)
	tests.AssertNoError(t, err, "Should add second subscription")

	subs1 := mgr1.GetAllSubscriptions()
	tests.AssertEqual(t, 2, len(subs1), "Should have 2 subscriptions")

	// Create new manager instance (simulating app restart)
	mgr2 := services.NewSubscriptionManager(tmpDir)
	subs2 := mgr2.GetAllSubscriptions()

	// Verify subscriptions were persisted
	tests.AssertEqual(t, 2, len(subs2), "Should have 2 subscriptions after reload")
	tests.AssertEqual(t, url1, subs2[0].URL, "First subscription URL should match")
	tests.AssertEqual(t, url2, subs2[1].URL, "Second subscription URL should match")
	tests.AssertEqual(t, "Subscription 1", subs2[0].Name, "First subscription name should match")
	tests.AssertEqual(t, "Subscription 2", subs2[1].Name, "Second subscription name should match")

	t.Logf("✓ Subscription persistence test passed")
}

// TestIntegrationSubscriptionDuplicatePrevention tests that duplicate subscriptions are prevented
func TestIntegrationSubscriptionDuplicatePrevention(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	subscriptionMgr := services.NewSubscriptionManager(tmpDir)

	url := "http://example.com/playlist.m3u"

	// Add subscription
	err := subscriptionMgr.AddSubscription(url, "Subscription 1", true)
	tests.AssertNoError(t, err, "Should add subscription successfully")

	// Try to add duplicate subscription
	err = subscriptionMgr.AddSubscription(url, "Subscription 1 Duplicate", true)
	tests.AssertError(t, err, "Should reject duplicate subscription")

	// Verify only one subscription exists
	subs := subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have only 1 subscription")

	t.Logf("✓ Subscription duplicate prevention test passed")
}

// TestIntegrationSubscriptionConcurrentOperations tests concurrent subscription operations
func TestIntegrationSubscriptionConcurrentOperations(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	subscriptionMgr := services.NewSubscriptionManager(tmpDir)

	// Add multiple subscriptions concurrently
	done := make(chan bool, 5)

	for i := 1; i <= 5; i++ {
		go func(index int) {
			url := "http://example.com/playlist" + string(rune(index)) + ".m3u"
			name := "Subscription " + string(rune(index))
			err := subscriptionMgr.AddSubscription(url, name, true)
			tests.AssertNoError(t, err, "Should add subscription successfully")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all subscriptions were added
	subs := subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 5, len(subs), "Should have 5 subscriptions")

	t.Logf("✓ Subscription concurrent operations test passed")
}
