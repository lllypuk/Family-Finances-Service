package integration_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

func TestReportHandler_Integration(t *testing.T) {
	t.Run("CreateReport_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
			StartDate: startDate,
			EndDate:   endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.ReportResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Monthly Expense Report", response.Data.Name)
		assert.Equal(t, "expenses", response.Data.Type)
		assert.Equal(t, testServer.AuthUser.ID, response.Data.UserID)

		stored, err := testServer.Repos.Report.GetByID(context.Background(), response.Data.ID)
		require.NoError(t, err)
		assert.Equal(t, response.Data.ID, stored.ID)
	})

	t.Run("CreateReport_ValidationError", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "name",
			},
			{
				name: "invalid_type",
				request: handlers.CreateReportRequest{
					Name:      "Test Report",
					Type:      "invalid_type",
					Period:    "monthly",
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "type",
			},
			{
				name: "invalid_period",
				request: handlers.CreateReportRequest{
					Name:      "Test Report",
					Type:      "expenses",
					Period:    "invalid_period",
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				},
				field: "period",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				testServer.Auth(t).Apply(req)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

				var response handlers.ErrorResponse
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.Error.Details)
				found := false
				for _, validationError := range response.Error.Details {
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
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
		assert.Equal(t, testReport.UserID, response.Data.UserID)
		assert.NotNil(t, response.Data.Data)
	})

	t.Run("GetReportByID_NotFound", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/%s", nonExistentID), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetReportByID_InvalidUUID", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/invalid-uuid", nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetReports_ByFamily", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
	})

	t.Run("GetReports_ByUser", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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

	// Test removed: GetReports_MissingFamilyID - no longer relevant in single-family model

	t.Run("DeleteReport_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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

		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/api/v1/reports/%s?family_id=%s", testReport.ID, family.ID),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify report is deleted by trying to get it
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/%s", testReport.ID), nil)
		testServer.Auth(t).Apply(getReq)
		getRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusNotFound, getRec.Code)
	})

	t.Run("CreateReport_DifferentTypes", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		reportTypes := []string{"expenses", "income", "budget", "cash_flow", "category_breakdown"}

		for _, reportType := range reportTypes {
			t.Run(fmt.Sprintf("type_%s", reportType), func(t *testing.T) {
				request := handlers.CreateReportRequest{
					Name:      fmt.Sprintf("Test %s Report", reportType),
					Type:      reportType,
					Period:    "monthly",
					StartDate: time.Now().AddDate(0, -1, 0),
					EndDate:   time.Now(),
				}

				requestBodyBytes, err := json.Marshal(request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				testServer.Auth(t).Apply(req)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusCreated, rec.Code)

				var response handlers.APIResponse[handlers.ReportResponse]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, response.Data.ID)
			})
		}
	})

	t.Run("CreateReport_DifferentPeriods", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
					StartDate: startDate,
					EndDate:   endDate,
				}

				requestBodyBytes, err := json.Marshal(request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(requestBodyBytes))
				testServer.Auth(t).Apply(req)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusCreated, rec.Code)

				var response handlers.APIResponse[handlers.ReportResponse]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, response.Data.ID)
			})
		}
	})
}

// TestReportAPI_GenerateAndExport — генерация отчёта через API и выгрузка его в CSV.
func TestReportAPI_GenerateAndExport(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	cat := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, cat))

	tx := testhelpers.CreateTestTransaction(
		testServer.AuthFamily.ID, testServer.AuthUser.ID, cat.ID, transaction.TypeExpense,
	)
	tx.Amount = 150.5
	tx.Date = time.Now().AddDate(0, 0, -1)
	require.NoError(t, testServer.Repos.Transaction.Create(ctx, tx))

	request := handlers.CreateReportRequest{
		Name:      "Export Report",
		Type:      "expenses",
		Period:    "monthly",
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(body))
	session.Apply(createReq)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code, createRec.Body.String())

	var created handlers.APIResponse[handlers.ReportResponse]
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.Data.ID.String(), nil)
	session.Apply(getReq)
	getRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)

	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.Data.ID.String()+"/export", nil)
	session.Apply(exportReq)
	exportRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(exportRec, exportReq)

	require.Equal(t, http.StatusOK, exportRec.Code, exportRec.Body.String())
	assert.Contains(t, exportRec.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, exportRec.Header().Get("Content-Disposition"), "attachment")

	csvBody := strings.TrimPrefix(exportRec.Body.String(), "\ufeff")
	rows, err := csv.NewReader(strings.NewReader(csvBody)).ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, []string{"Category", "Amount", "Percentage", "Transaction Count"}, rows[0])
	assert.Equal(t, "TOTAL", rows[len(rows)-1][0])
}

// TestReportAPI_CreateReport_CustomPeriodKeepsDates — при period=custom границы берутся из запроса,
// а не из календарного месяца (иначе category_breakdown/budget молча считают текущий месяц).
func TestReportAPI_CreateReport_CustomPeriodKeepsDates(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

	request := handlers.CreateReportRequest{
		Name:      "Custom Breakdown",
		Type:      "category_breakdown",
		Period:    "custom",
		StartDate: start,
		EndDate:   end,
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(body))
	session.Apply(req)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created handlers.APIResponse[handlers.ReportResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.True(t, created.Data.StartDate.Equal(start), "start: %s", created.Data.StartDate)
	assert.True(t, created.Data.EndDate.Equal(end), "end: %s", created.Data.EndDate)
}

// TestStatsAPI_Summary — сводка за период совпадает с созданными операциями.
func TestStatsAPI_Summary(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	expenseCat := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, expenseCat))

	expense := testhelpers.CreateTestTransaction(
		testServer.AuthFamily.ID, testServer.AuthUser.ID, expenseCat.ID, transaction.TypeExpense,
	)
	expense.Amount = 200
	expense.Date = time.Now()
	require.NoError(t, testServer.Repos.Transaction.Create(ctx, expense))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary", nil)
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.InDelta(t, 200.0, response.Data.Current.Expenses, 0.001)
	assert.Equal(t, 1, response.Data.Current.TransactionCount)
	require.Len(t, response.Data.ExpenseCategories, 1)
	assert.Equal(t, expenseCat.Name, response.Data.ExpenseCategories[0].Name)
}

// TestStatsAPI_Summary_InvalidDate — нераспознанная дата отбивается 422, сервис не вызывается.
func TestStatsAPI_Summary_InvalidDate(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?from=01.03.2025", nil)
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "from", response.Error.Details[0].Field)
}

// Перевёрнутый диапазон дат раньше давал 201 и пустой отчёт.
func TestReportAPI_CreateReport_RejectsInvertedDateRange(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	body := mustJSON(t, map[string]any{
		"name":       "Inverted",
		"type":       "expenses",
		"period":     "monthly",
		"start_date": "2026-03-31T00:00:00Z",
		"end_date":   "2026-03-01T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	testServer.Auth(t).Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.NotEmpty(t, response.Error.Details)
	assert.Equal(t, "end_date", response.Error.Details[0].Field)
}
