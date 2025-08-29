package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/testhelpers"
)

func TestReportHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateReport_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		startDate := time.Now().AddDate(0, -1, 0) // one month ago
		endDate := time.Now()

		request := handlers.CreateReportRequest{
			Name:      "Monthly Expense Report",
			Type:      "expenses",
			Period:    "monthly",
			FamilyID:  family.ID,
			UserID:    user.ID,
			StartDate: startDate,
			EndDate:   endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.ReportResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Name, response.Data.Name)
		assert.Equal(t, request.Type, response.Data.Type)
		assert.Equal(t, request.Period, response.Data.Period)
		assert.Equal(t, request.FamilyID, response.Data.FamilyID)
		assert.Equal(t, request.UserID, response.Data.UserID)
		assert.NotNil(t, response.Data.Data)
		assert.NotZero(t, response.Data.ID)
		assert.NotZero(t, response.Data.GeneratedAt)
	})

	t.Run("CreateReport_ValidationError", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		tests := []struct {
			name    string
			request handlers.CreateReportRequest
			field   string
		}{
			{
				name: "empty_name",
				request: handlers.CreateReportRequest{
					Name:      "",
					Type:      "expenses",
					Period:    "monthly",
					FamilyID:  family.ID,
					UserID:    user.ID,
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "Name",
			},
			{
				name: "invalid_type",
				request: handlers.CreateReportRequest{
					Name:      "Test Report",
					Type:      "invalid_type",
					Period:    "monthly",
					FamilyID:  family.ID,
					UserID:    user.ID,
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "Type",
			},
			{
				name: "invalid_period",
				request: handlers.CreateReportRequest{
					Name:      "Test Report",
					Type:      "expenses",
					Period:    "invalid_period",
					FamilyID:  family.ID,
					UserID:    user.ID,
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "Period",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)

				var response handlers.APIResponse[any]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.Errors)
				found := false
				for _, validationError := range response.Errors {
					if validationError.Field == tt.field {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected validation error for field %s", tt.field)
			})
		}
	})

	t.Run("GetReportByID_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testReport := testhelpers.CreateTestReport(family.ID, user.ID)
		err = testServer.Repos.Report.Create(context.Background(), testReport)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/%s", testReport.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.ReportResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testReport.ID, response.Data.ID)
		assert.Equal(t, testReport.Name, response.Data.Name)
		assert.Equal(t, string(testReport.Type), response.Data.Type)
		assert.Equal(t, string(testReport.Period), response.Data.Period)
		assert.Equal(t, testReport.FamilyID, response.Data.FamilyID)
		assert.Equal(t, testReport.UserID, response.Data.UserID)
		assert.NotNil(t, response.Data.Data)
	})

	t.Run("GetReportByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/%s", nonExistentID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetReportByID_InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/invalid-uuid", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetReports_ByFamily", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Create test reports
		report1 := testhelpers.CreateTestReport(family.ID, user.ID)
		report1.Name = "Report 1"
		report1.Type = report.TypeExpenses

		report2 := testhelpers.CreateTestReport(family.ID, user.ID)
		report2.Name = "Report 2"
		report2.Type = report.TypeIncome

		err = testServer.Repos.Report.Create(context.Background(), report1)
		require.NoError(t, err)
		err = testServer.Repos.Report.Create(context.Background(), report2)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports?family_id=%s", family.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.ReportResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2)

		reportIDs := []uuid.UUID{response.Data[0].ID, response.Data[1].ID}
		assert.Contains(t, reportIDs, report1.ID)
		assert.Contains(t, reportIDs, report2.ID)

		for _, report := range response.Data {
			assert.Equal(t, family.ID, report.FamilyID)
		}
	})

	t.Run("GetReports_ByUser", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user1 := testhelpers.CreateTestUser(family.ID)
		user1.Email = "user1@example.com"
		user2 := testhelpers.CreateTestUser(family.ID)
		user2.Email = "user2@example.com"

		err = testServer.Repos.User.Create(context.Background(), user1)
		require.NoError(t, err)
		err = testServer.Repos.User.Create(context.Background(), user2)
		require.NoError(t, err)

		// Create reports for different users
		reportUser1 := testhelpers.CreateTestReport(family.ID, user1.ID)
		reportUser2 := testhelpers.CreateTestReport(family.ID, user2.ID)

		err = testServer.Repos.Report.Create(context.Background(), reportUser1)
		require.NoError(t, err)
		err = testServer.Repos.Report.Create(context.Background(), reportUser2)
		require.NoError(t, err)

		// Get reports for user1 only
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/reports?family_id=%s&user_id=%s", family.ID, user1.ID),
			nil,
		)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.ReportResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, reportUser1.ID, response.Data[0].ID)
		assert.Equal(t, user1.ID, response.Data[0].UserID)
	})

	t.Run("GetReports_MissingFamilyID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("DeleteReport_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testReport := testhelpers.CreateTestReport(family.ID, user.ID)
		err = testServer.Repos.Report.Create(context.Background(), testReport)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/reports/%s", testReport.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify report is deleted by trying to get it
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/%s", testReport.ID), nil)
		getRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusNotFound, getRec.Code)
	})

	t.Run("CreateReport_DifferentTypes", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		reportTypes := []string{"expenses", "income", "budget", "cash_flow", "category_break"}

		for _, reportType := range reportTypes {
			t.Run(fmt.Sprintf("type_%s", reportType), func(t *testing.T) {
				request := handlers.CreateReportRequest{
					Name:      fmt.Sprintf("Test %s Report", reportType),
					Type:      reportType,
					Period:    "monthly",
					FamilyID:  family.ID,
					UserID:    user.ID,
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				}

				requestBodyBytes, err := json.Marshal(request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusCreated, rec.Code)

				var response handlers.APIResponse[handlers.ReportResponse]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, reportType, response.Data.Type)
				assert.NotNil(t, response.Data.Data)
			})
		}
	})

	t.Run("CreateReport_DifferentPeriods", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		periods := []string{"daily", "weekly", "monthly", "yearly", "custom"}

		for _, period := range periods {
			t.Run(fmt.Sprintf("period_%s", period), func(t *testing.T) {
				var startDate, endDate time.Time

				switch period {
				case "daily":
					startDate = time.Now().Truncate(24 * time.Hour)
					endDate = startDate.Add(24 * time.Hour)
				case "weekly":
					startDate = time.Now().AddDate(0, 0, -7)
					endDate = time.Now()
				case "monthly":
					startDate = time.Now().AddDate(0, -1, 0)
					endDate = time.Now()
				case "yearly":
					startDate = time.Now().AddDate(-1, 0, 0)
					endDate = time.Now()
				case "custom":
					startDate = time.Now().AddDate(0, -2, 0)
					endDate = time.Now().AddDate(0, -1, 0)
				}

				request := handlers.CreateReportRequest{
					Name:      fmt.Sprintf("Test %s Report", period),
					Type:      "expenses",
					Period:    period,
					FamilyID:  family.ID,
					UserID:    user.ID,
					StartDate: startDate,
					EndDate:   endDate,
				}

				requestBodyBytes, err := json.Marshal(request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusCreated, rec.Code)

				var response handlers.APIResponse[handlers.ReportResponse]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, period, response.Data.Period)
			})
		}
	})
}
