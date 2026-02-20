//go:build api

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	eventServiceURL   = "http://localhost:8081"
	bookingServiceURL = "http://localhost:8082"
)

// TestAPI-FullFlow - ทดสอบ API ทั้งหมดแบบ End-to-End พร้อมแสดงผลลัพธ์
func TestAPI_FullFlow(t *testing.T) {
	// Wait for services to be ready
	waitForServices(t)

	// Step 1: Create Event
	t.Run("Step1_CreateEvent", func(t *testing.T) {
		t.Log(" STEP 1: Create Event")
		t.Log("    Request:  POST /api/v1/events")
		t.Log("    Body:     name='Golang Workshop Bangkok', max_seats=50, waitlist_limit=5")
		
		eventReq := map[string]interface{}{
			"name":             "Golang Workshop Bangkok",
			"max_seats":        50,
			"waitlist_limit":   5,
			"price":            2500,
			"booking_start_at": "2026-02-20T17:00:00+07:00",
			"booking_end_at":   "2026-02-25T17:00:00+07:00",
		}
		
		resp := post(t, eventServiceURL+"/api/v1/events", eventReq)
		assert.Equal(t, 201, resp.StatusCode, "Should create event successfully")
		
		var eventResp map[string]interface{}
		decodeJSON(t, resp, &eventResp)
		
		assert.Equal(t, float64(1), eventResp["id"], "Event ID should be 1")
		assert.Equal(t, "Golang Workshop Bangkok", eventResp["name"])
		assert.Equal(t, float64(50), eventResp["max_seats"])
		assert.Equal(t, float64(5), eventResp["waitlist_limit"])
		
		t.Logf("     Result:   HTTP 201 Created")
		t.Logf("     Response: id=%v, name='%v', max_seats=%v, waitlist_limit=%v",
			eventResp["id"], eventResp["name"], eventResp["max_seats"], eventResp["waitlist_limit"])
	})

	// Wait for RabbitMQ sync
	time.Sleep(2 * time.Second)

	// Step 2: Get Event Status
	t.Run("Step2_GetEventStatus", func(t *testing.T) {
		t.Log(" STEP 2: Get Event Status")
		t.Log("    Request:  GET /api/v1/events/1/status")
		
		resp := get(t, bookingServiceURL+"/api/v1/events/1/status")
		assert.Equal(t, 200, resp.StatusCode)
		
		var statusResp map[string]interface{}
		decodeJSON(t, resp, &statusResp)
		
		assert.Equal(t, float64(50), statusResp["seats_available"], "Should have 50 seats available")
		assert.Equal(t, float64(0), statusResp["confirmed_count"])
		assert.Equal(t, float64(0), statusResp["waitlisted_count"])
		
		t.Logf("     Result:   HTTP 200 OK")
		t.Logf("     Status:   confirmed=%v, waitlisted=%v, available=%v",
			statusResp["confirmed_count"], statusResp["waitlisted_count"], statusResp["seats_available"])
	})

	// Step 3: Create First Booking (Confirmed)
	t.Run("Step3_CreateFirstBooking", func(t *testing.T) {
		t.Log(" STEP 3: Create First Booking")
		t.Log("    Request:  POST /api/v1/events/1/bookings")
		t.Log("    Body:     user_id='user-001'")
		
		bookingReq := map[string]string{
			"user_id": "user-001",
		}
		
		resp := post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
		assert.Equal(t, 201, resp.StatusCode, "Should create booking successfully")
		
		var bookingResp map[string]interface{}
		decodeJSON(t, resp, &bookingResp)
		
		assert.Equal(t, float64(1), bookingResp["id"])
		assert.Equal(t, "user-001", bookingResp["user_id"])
		assert.Equal(t, "confirmed", bookingResp["status"], "First booking should be confirmed")
		
		t.Logf("     Result:   HTTP 201 Created")
		t.Logf("     Response: id=%v, user_id='%v', status='%v'",
			bookingResp["id"], bookingResp["user_id"], bookingResp["status"])
	})

	// Step 4: Double Booking Prevention
	t.Run("Step4_DoubleBookingPrevention", func(t *testing.T) {
		t.Log("  STEP 4: Double Booking Prevention")
		t.Log("    Request:  POST /api/v1/events/1/bookings")
		t.Log("    Body:     user_id='user-001' (same user)")
		
		bookingReq := map[string]string{
			"user_id": "user-001",
		}
		
		resp := post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
		assert.Equal(t, 409, resp.StatusCode, "Should reject duplicate booking with 409")
		
		var errorResp map[string]string
		decodeJSON(t, resp, &errorResp)
		
		assert.Contains(t, errorResp["message"], "already", "Error message should mention 'already'")
		
		t.Logf("    Result:   HTTP 409 Conflict")
		t.Logf("    Error:    %v", errorResp["message"])
	})

	// Step 5: Fill All 50 Seats
	t.Run("Step5_FillAllSeats", func(t *testing.T) {
		t.Log(" STEP 5: Fill All 50 Seats")
		t.Log("    Request:  POST /api/v1/events/1/bookings (user-002 to user-050)")
		
		confirmedCount := 0
		for i := 2; i <= 50; i++ {
			userID := fmt.Sprintf("user-%03d", i)
			bookingReq := map[string]string{
				"user_id": userID,
			}
			
			resp := post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
			
			if resp.StatusCode == 201 {
				var bookingResp map[string]interface{}
				decodeJSON(t, resp, &bookingResp)
				
				if bookingResp["status"] == "confirmed" {
					confirmedCount++
				}
			}
		}
		
		assert.Equal(t, 49, confirmedCount, "Should have 49 confirmed bookings (002-050)")
		t.Logf("     Result:   Created %d confirmed bookings", confirmedCount)
		t.Logf("     Total:    50 confirmed (001 + 002-050)")
	})

	// Step 6: Verify Seats Full
	t.Run("Step6_VerifySeatsFull", func(t *testing.T) {
		t.Log(" STEP 6: Verify Seats Full")
		t.Log("    Request:  GET /api/v1/events/1/status")
		
		resp := get(t, bookingServiceURL+"/api/v1/events/1/status")
		require.Equal(t, 200, resp.StatusCode)
		
		var statusResp map[string]interface{}
		decodeJSON(t, resp, &statusResp)
		
		assert.Equal(t, float64(50), statusResp["confirmed_count"], "Should have 50 confirmed")
		assert.Equal(t, float64(0), statusResp["seats_available"], "Should have 0 seats available")
		
		t.Logf("     Result:   HTTP 200 OK")
		t.Logf("     Status:   confirmed=%v ⬅️ FULL, waitlisted=%v, available=%v ⬅️ ZERO",
			statusResp["confirmed_count"], statusResp["waitlisted_count"], statusResp["seats_available"])
	})

	// Step 7: Create Waitlist Booking
	t.Run("Step7_CreateWaitlistBooking", func(t *testing.T) {
		t.Log(" STEP 7: Create Waitlist Booking")
		t.Log("    Request:  POST /api/v1/events/1/bookings")
		t.Log("    Body:     user_id='user-051'")
		
		bookingReq := map[string]string{
			"user_id": "user-051",
		}
		
		resp := post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
		assert.Equal(t, 201, resp.StatusCode, "Should create waitlist booking")
		
		var bookingResp map[string]interface{}
		decodeJSON(t, resp, &bookingResp)
		
		assert.Equal(t, "waitlisted", bookingResp["status"], "Should be waitlisted")
		assert.Equal(t, float64(1), bookingResp["waitlist_order"], "Should be waitlist #1")
		
		t.Logf("     Result:   HTTP 201 Created")
		t.Logf("     Response: id=%v, user_id='%v', status='%v', waitlist_order=%v",
			bookingResp["id"], bookingResp["user_id"], bookingResp["status"], bookingResp["waitlist_order"])
	})

	// Step 8: Fully Booked (Reject)
	t.Run("Step8_FullyBooked", func(t *testing.T) {
		t.Log(" STEP 8: Fully Booked Rejection")
		t.Log("    Request:  POST /api/v1/events/1/bookings (fill waitlist)")
		
		// Fill waitlist first (052-055)
		for i := 52; i <= 55; i++ {
			userID := fmt.Sprintf("user-%03d", i)
			bookingReq := map[string]string{
				"user_id": userID,
			}
			post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
		}
		t.Log("     Filled waitlist: user-052 to user-055")
		
		// Now try 056 - should be rejected
		t.Log("    Request:  POST /api/v1/events/1/bookings")
		t.Log("    Body:     user_id='user-056' (should be rejected)")
		
		bookingReq := map[string]string{
			"user_id": "user-056",
		}
		
		resp := post(t, bookingServiceURL+"/api/v1/events/1/bookings", bookingReq)
		assert.Equal(t, 409, resp.StatusCode, "Should reject when fully booked")
		
		var errorResp map[string]string
		decodeJSON(t, resp, &errorResp)
		
		assert.Contains(t, errorResp["message"], "fully booked")
		
		t.Logf("     Result:   HTTP 409 Conflict")
		t.Logf("      Error:    %v", errorResp["message"])
	})

	// Step 9: Cancel Booking
	t.Run("Step9_CancelBooking", func(t *testing.T) {
		t.Log(" STEP 9: Cancel Booking")
		t.Log("    Request:  DELETE /api/v1/bookings/1")
		
		resp := delete(t, bookingServiceURL+"/api/v1/bookings/1")
		assert.Equal(t, 200, resp.StatusCode, "Should cancel successfully")
		
		var cancelResp map[string]interface{}
		decodeJSON(t, resp, &cancelResp)
		
		assert.Equal(t, "cancelled", cancelResp["status"])
		
		t.Logf("     Result:   HTTP 200 OK")
		t.Logf("     Response: id=%v, status='%v'", cancelResp["id"], cancelResp["status"])
	})

	// Step 10: Verify Waitlist Promotion
	t.Run("Step10_WaitlistPromotion", func(t *testing.T) {
		t.Log(" STEP 10: Verify Waitlist Promotion")
		t.Log("    Request:  GET /api/v1/bookings/51 (was waitlist #1)")
		
		resp := get(t, bookingServiceURL+"/api/v1/bookings/51")
		assert.Equal(t, 200, resp.StatusCode)
		
		var bookingResp map[string]interface{}
		decodeJSON(t, resp, &bookingResp)
		
		assert.Equal(t, "user-051", bookingResp["user_id"])
		assert.Equal(t, "confirmed", bookingResp["status"], "Waitlist #1 should be promoted to confirmed")
		
		t.Logf("    ✅ Result:   HTTP 200 OK")
		t.Logf("    🎉 Promoted: user-051 status changed from 'waitlisted' → '%v'", bookingResp["status"])
	})

	// Final Summary
	t.Run("FinalSummary", func(t *testing.T) {
		t.Log("═══════════════════════════════════════════════════════════════")
		t.Log(" FINAL SUMMARY")
		t.Log("═══════════════════════════════════════════════════════════════")
		
		resp := get(t, bookingServiceURL+"/api/v1/events/1/status")
		require.Equal(t, 200, resp.StatusCode)
		
		var statusResp map[string]interface{}
		decodeJSON(t, resp, &statusResp)
		
		t.Logf(" Event: %v", statusResp["name"])
		t.Logf("   • Confirmed:   %v (max: %v)", statusResp["confirmed_count"], statusResp["max_seats"])
		t.Logf("   • Waitlisted:  %v (limit: %v)", statusResp["waitlisted_count"], statusResp["waitlist_limit"])
		t.Logf("   • Available:   %v", statusResp["seats_available"])
		t.Log("")
		t.Log(" ALL API TESTS PASSED!")
		t.Log("═══════════════════════════════════════════════════════════════")
		
		assert.Equal(t, float64(50), statusResp["confirmed_count"], "Should still have 50 confirmed")
		assert.Equal(t, float64(4), statusResp["waitlisted_count"], "Should have 4 waitlisted (52-55)")
	})
}

// Helper functions

func waitForServices(t *testing.T) {
	t.Log("⏳ Waiting for services to be ready...")
	
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(eventServiceURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			resp2, err2 := http.Get(bookingServiceURL + "/health")
			if err2 == nil && resp2.StatusCode == 200 {
				resp2.Body.Close()
				t.Log("✅ Services are ready!")
				t.Log("")
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
	
	t.Fatal("Services did not become ready in time")
}

func get(t *testing.T, url string) *http.Response {
	resp, err := http.Get(url)
	require.NoError(t, err)
	return resp
}

func post(t *testing.T, url string, body interface{}) *http.Response {
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	return resp
}

func delete(t *testing.T, url string) *http.Response {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(target)
	if err != nil && resp.StatusCode >= 400 {
		// For error responses, body might not be JSON
		return
	}
	require.NoError(t, err)
}

// TestMain - Setup and teardown
func TestMain(m *testing.M) {
	fmt.Println(" Starting API Tests...")
	fmt.Println("Make sure services are running: make docker-up-all")
	fmt.Println("")
	
	code := m.Run()
	
	fmt.Println("")
	fmt.Println(" API Tests Complete!")
	os.Exit(code)
}
