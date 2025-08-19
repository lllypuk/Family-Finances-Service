package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// TestAuthenticationFlow tests complete authentication workflow
func TestAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e auth flow test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080"
	apiURL := baseURL + "/api/v1"

	// Create HTTP client with cookie jar for session management
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	var familyID string

	// Setup test data
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Auth Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(apiURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create admin user
		userData := map[string]any{
			"email":      "admin@auth-test.com",
			"password":   "SecurePassword123!",
			"first_name": "Admin",
			"last_name":  "User",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(apiURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		_ = extractIDFromResponse(userResponse) // adminUserID not used elsewhere
	})

	t.Run("WebAuthenticationFlow", func(t *testing.T) {
		// Step 1: Access login page
		t.Run("AccessLoginPage", func(t *testing.T) {
			resp, err := client.Get(baseURL + "/login")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check for login form elements (basic check)
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			content := string(body[:n])

			assert.Contains(t, content, "login", "Login page should contain login form")
		})

		// Step 2: Attempt login with invalid credentials
		t.Run("InvalidLogin", func(t *testing.T) {
			loginData := url.Values{
				"email":    {"admin@auth-test.com"},
				"password": {"WrongPassword"},
			}

			resp, err := client.PostForm(baseURL+"/login", loginData)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should redirect back to login with error or show error
			assert.True(t, resp.StatusCode >= 400 || resp.StatusCode == http.StatusFound,
				"Invalid login should result in error or redirect")
		})

		// Step 3: Successful login
		t.Run("ValidLogin", func(t *testing.T) {
			loginData := url.Values{
				"email":    {"admin@auth-test.com"},
				"password": {"SecurePassword123!"},
			}

			resp, err := client.PostForm(baseURL+"/login", loginData)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should redirect to dashboard or success page
			assert.True(t, resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusOK,
				"Valid login should succeed")

			// Check if session cookie is set
			u, _ := url.Parse(baseURL)
			cookies := client.Jar.Cookies(u)

			sessionFound := false
			for _, cookie := range cookies {
				if strings.Contains(cookie.Name, "session") || strings.Contains(cookie.Name, "auth") {
					sessionFound = true
					break
				}
			}
			assert.True(t, sessionFound, "Session cookie should be set after login")
		})

		// Step 4: Access protected page
		t.Run("AccessProtectedPage", func(t *testing.T) {
			resp, err := client.Get(baseURL + "/dashboard")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"Authenticated user should access dashboard")
		})

		// Step 5: Logout
		t.Run("Logout", func(t *testing.T) {
			resp, err := client.Post(baseURL+"/logout", "application/json", nil)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should redirect to login page or home
			assert.True(t, resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusOK,
				"Logout should succeed")
		})

		// Step 6: Verify logout worked
		t.Run("VerifyLogout", func(t *testing.T) {
			resp, err := client.Get(baseURL + "/dashboard")
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should redirect to login or return unauthorized
			assert.True(t, resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusUnauthorized,
				"Should not access dashboard after logout")
		})
	})
}

// TestSessionManagement tests session security and management
func TestSessionManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e session management test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080"
	apiURL := baseURL + "/api/v1"

	var familyID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Session Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(apiURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create user
		userData := map[string]any{
			"email":      "session@test.com",
			"password":   "SessionPassword123!",
			"first_name": "Session",
			"last_name":  "Test",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(apiURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		_ = extractIDFromResponse(userResponse) // userID not used elsewhere
	})

	t.Run("ConcurrentSessions", func(t *testing.T) {
		// Create two separate clients with different cookie jars
		jar1, _ := cookiejar.New(nil)
		jar2, _ := cookiejar.New(nil)
		client1 := &http.Client{Jar: jar1}
		client2 := &http.Client{Jar: jar2}

		// Login with both clients
		loginData := url.Values{
			"email":    {"session@test.com"},
			"password": {"SessionPassword123!"},
		}

		// Client 1 login
		resp1, err := client1.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp1.Body.Close()

		// Client 2 login
		resp2, err := client2.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp2.Body.Close()

		// Both should be able to access protected resources
		resp1, err = client1.Get(baseURL + "/dashboard")
		require.NoError(t, err)
		defer resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		resp2, err = client2.Get(baseURL + "/dashboard")
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("SessionTimeout", func(t *testing.T) {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}

		// Login
		loginData := url.Values{
			"email":    {"session@test.com"},
			"password": {"SessionPassword123!"},
		}

		resp, err := client.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Access protected resource
		resp, err = client.Get(baseURL + "/dashboard")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Note: In a real test, you might wait for session timeout
		// or manipulate session expiry time for testing
		// For now, we just verify the session works immediately after login
	})
}

// TestRoleBasedAccess tests role-based access control
func TestRoleBasedAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e role-based access test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080"
	apiURL := baseURL + "/api/v1"

	var familyID string
	var adminUserID, memberUserID, childUserID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Role Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(apiURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create users with different roles
		users := []map[string]any{
			{
				"email":      "admin@role-test.com",
				"password":   "AdminPass123!",
				"first_name": "Admin",
				"last_name":  "User",
				"role":       "admin",
				"family_id":  familyID,
			},
			{
				"email":      "member@role-test.com",
				"password":   "MemberPass123!",
				"first_name": "Member",
				"last_name":  "User",
				"role":       "member",
				"family_id":  familyID,
			},
			{
				"email":      "child@role-test.com",
				"password":   "ChildPass123!",
				"first_name": "Child",
				"last_name":  "User",
				"role":       "child",
				"family_id":  familyID,
			},
		}

		userIDs := []*string{&adminUserID, &memberUserID, &childUserID}

		for i, userData := range users {
			body, _ := json.Marshal(userData)
			resp, err := http.Post(apiURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			var userResponse map[string]any
			err = json.NewDecoder(resp.Body).Decode(&userResponse)
			require.NoError(t, err)
			*userIDs[i] = extractIDFromResponse(userResponse)
		}
	})

	t.Run("AdminAccess", func(t *testing.T) {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}

		// Login as admin
		loginData := url.Values{
			"email":    {"admin@role-test.com"},
			"password": {"AdminPass123!"},
		}

		resp, err := client.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Admin should access all areas
		protectedPages := []string{
			"/dashboard",
			"/users",
			"/budgets",
			"/reports",
			"/settings",
		}

		for _, page := range protectedPages {
			resp, err := client.Get(baseURL + page)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
				"Admin should access %s (got %d)", page, resp.StatusCode)
		}
	})

	t.Run("MemberAccess", func(t *testing.T) {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}

		// Login as member
		loginData := url.Values{
			"email":    {"member@role-test.com"},
			"password": {"MemberPass123!"},
		}

		resp, err := client.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Member should access most areas but not admin settings
		resp, err = client.Get(baseURL + "/dashboard")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Member should access dashboard")

		resp, err = client.Get(baseURL + "/budgets")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Member should access budgets")
	})

	t.Run("ChildAccess", func(t *testing.T) {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}

		// Login as child
		loginData := url.Values{
			"email":    {"child@role-test.com"},
			"password": {"ChildPass123!"},
		}

		resp, err := client.PostForm(baseURL+"/login", loginData)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Child should have limited access
		resp, err = client.Get(baseURL + "/dashboard")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Child should access basic dashboard")

		// Child should not access admin areas (if implemented)
		resp, err = client.Get(baseURL + "/admin/users")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound,
			"Child should not access admin areas")
	})
}

// TestPasswordSecurity tests password security requirements
func TestPasswordSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e password security test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	apiURL := "http://localhost:8080/api/v1"

	var familyID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		familyData := map[string]any{
			"name":     "Password Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(apiURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)
	})

	t.Run("WeakPasswordRejection", func(t *testing.T) {
		weakPasswords := []string{
			"123",      // Too short
			"password", // Common password
			"12345678", // Numbers only
			"abcdefgh", // Letters only
			"Password", // No numbers/special chars
		}

		for i, weakPassword := range weakPasswords {
			userData := map[string]any{
				"email":      fmt.Sprintf("weak%d@test.com", i),
				"password":   weakPassword,
				"first_name": "Test",
				"last_name":  "User",
				"role":       "member",
				"family_id":  familyID,
			}

			body, _ := json.Marshal(userData)
			resp, err := http.Post(apiURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			// Weak passwords should be rejected
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"Weak password '%s' should be rejected", weakPassword)
		}
	})

	t.Run("StrongPasswordAcceptance", func(t *testing.T) {
		strongPasswords := []string{
			"StrongPass123!",
			"MySecure@Password2024",
			"Complex!Password99",
		}

		for i, strongPassword := range strongPasswords {
			userData := map[string]any{
				"email":      fmt.Sprintf("strong%d@test.com", i),
				"password":   strongPassword,
				"first_name": "Test",
				"last_name":  "User",
				"role":       "member",
				"family_id":  familyID,
			}

			body, _ := json.Marshal(userData)
			resp, err := http.Post(apiURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			// Strong passwords should be accepted
			assert.Equal(t, http.StatusCreated, resp.StatusCode,
				"Strong password '%s' should be accepted", strongPassword)
		}
	})
}
