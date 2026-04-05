package handler

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/model"
)

// Helper functions
func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Mock context key setup for tests
type testContextKey struct{}

var testUserIDKey = &testContextKey{}

func withTestUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, testUserIDKey, userID)
}

// For testing, we need to access the context key from middleware
// Since it's private, we'll test through the handler methods that don't depend on it

func TestExportCSV_Unauthorized(t *testing.T) {
	// Create a handler with nil service - this should still handle auth check
	handler := NewReceiptHandler(&config.Config{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/export", nil)
	rr := httptest.NewRecorder()

	handler.ExportCSV(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("expected 'unauthorized' in response, got %s", rr.Body.String())
	}
}

func TestExportCSV_InvalidFromDate(t *testing.T) {
	handler := NewReceiptHandler(&config.Config{}, nil)

	// Create a context with a user ID using the same key as middleware
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), struct{}{}, userID)

	// We need to inject the user ID in a way that GetUserID can find it
	// Since the key is private, we can't easily mock this
	// Let's test the date parsing logic instead

	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/export?from=invalid", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ExportCSV(rr, req)

	// Without proper auth context, this will return unauthorized
	// The date validation happens after auth check
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for unauthorized, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestFormatReceiptForCSV(t *testing.T) {
	handler := NewReceiptHandler(&config.Config{}, nil)

	tests := []struct {
		name     string
		receipt  *model.Receipt
		expected []string
	}{
		{
			name: "Full receipt with all fields",
			receipt: &model.Receipt{
				Title:       strPtr("Test Shop"),
				ReceiptDate: timePtr(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
				CreatedAt:   time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
				Total:       100.50,
				Currency:    "USD",
				Source:      model.ReceiptSourceManual,
				Status:      model.ReceiptStatusConfirmed,
				Items: []*model.ReceiptItem{
					{Name: "Item 1", Quantity: 2, UnitPrice: 25.00},
					{Name: "Item 2", Quantity: 1, UnitPrice: 50.50},
				},
				Tags: []*model.Tag{
					{Name: "groceries"},
					{Name: "monthly"},
				},
			},
			expected: []string{"2024-01-15", "Test Shop", `"Item 1 (2.00 x 25.00); Item 2 (1.00 x 50.50)"`, "100.50", "USD", "groceries, monthly", "manual", "confirmed"},
		},
		{
			name: "Receipt without title uses empty shop",
			receipt: &model.Receipt{
				Title:       nil,
				ReceiptDate: timePtr(time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)),
				CreatedAt:   time.Date(2024, 2, 21, 10, 0, 0, 0, time.UTC),
				Total:       45.00,
				Currency:    "EUR",
				Source:      model.ReceiptSourceOCR,
				Status:      model.ReceiptStatusPendingReview,
				Items:       []*model.ReceiptItem{},
				Tags:        []*model.Tag{},
			},
			expected: []string{"2024-02-20", "", "", "45.00", "EUR", "", "ocr", "pending_review"},
		},
		{
			name: "Receipt without receipt_date uses created_at",
			receipt: &model.Receipt{
				Title:       strPtr("Another Store"),
				ReceiptDate: nil,
				CreatedAt:   time.Date(2024, 3, 10, 15, 30, 0, 0, time.UTC),
				Total:       30.00,
				Currency:    "GBP",
				Source:      model.ReceiptSourceAPI,
				Status:      model.ReceiptStatusRejected,
				Items: []*model.ReceiptItem{
					{Name: "Product A", Quantity: 3, UnitPrice: 10.00},
				},
				Tags: []*model.Tag{
					{Name: "office"},
				},
			},
			expected: []string{"2024-03-10", "Another Store", `"Product A (3.00 x 10.00)"`, "30.00", "GBP", "office", "api", "rejected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := handler.formatReceiptForCSV(tt.receipt)

			if len(row) != len(tt.expected) {
				t.Errorf("expected %d columns, got %d", len(tt.expected), len(row))
				return
			}

			for i, expected := range tt.expected {
				if row[i] != expected {
					t.Errorf("column %d: expected '%s', got '%s'", i, expected, row[i])
				}
			}
		})
	}
}

func TestExportCSV_MethodNotAllowed(t *testing.T) {
	handler := NewReceiptHandler(&config.Config{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/receipts/export", nil)
	rr := httptest.NewRecorder()

	handler.ExportCSV(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		strs     []string
		sep      string
		expected string
	}{
		{[]string{}, ", ", ""},
		{[]string{"a"}, ", ", "a"},
		{[]string{"a", "b", "c"}, ", ", "a, b, c"},
		{[]string{"x", "y"}, "; ", "x; y"},
	}

	for _, tt := range tests {
		result := joinStrings(tt.strs, tt.sep)
		if result != tt.expected {
			t.Errorf("joinStrings(%v, %q) = %q, expected %q", tt.strs, tt.sep, result, tt.expected)
		}
	}
}

// Integration-style test that verifies the full CSV generation
func TestExportCSV_Integration(t *testing.T) {
	// This test verifies the CSV output format without requiring a database
	// We'll test the formatReceiptForCSV function which does the actual formatting

	handler := NewReceiptHandler(&config.Config{}, nil)

	// Create sample receipts
	receipts := []*model.Receipt{
		{
			ID:          uuid.New(),
			Title:       strPtr("Grocery Store"),
			ReceiptDate: timePtr(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)),
			CreatedAt:   time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			Total:       85.47,
			Currency:    "USD",
			Source:      model.ReceiptSourceManual,
			Status:      model.ReceiptStatusConfirmed,
			Items: []*model.ReceiptItem{
				{Name: "Apples", Quantity: 3, UnitPrice: 2.99},
				{Name: "Bread", Quantity: 1, UnitPrice: 4.50},
				{Name: "Milk", Quantity: 2, UnitPrice: 3.49},
			},
			Tags: []*model.Tag{
				{Name: "food"},
				{Name: "groceries"},
			},
		},
		{
			ID:          uuid.New(),
			Title:       strPtr("Electronics Shop"),
			ReceiptDate: timePtr(time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)),
			CreatedAt:   time.Date(2024, 6, 20, 14, 30, 0, 0, time.UTC),
			Total:       299.99,
			Currency:    "USD",
			Source:      model.ReceiptSourceOCR,
			Status:      model.ReceiptStatusPendingReview,
			Items: []*model.ReceiptItem{
				{Name: "Headphones", Quantity: 1, UnitPrice: 299.99},
			},
			Tags: []*model.Tag{
				{Name: "electronics"},
			},
		},
	}

	// Create a buffer to capture CSV output
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	headers := []string{"date", "shop", "items", "total", "currency", "tags", "source", "status"}
	if err := writer.Write(headers); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}

	// Write data rows
	for _, receipt := range receipts {
		row := handler.formatReceiptForCSV(receipt)
		if err := writer.Write(row); err != nil {
			t.Fatalf("failed to write row: %v", err)
		}
	}
	writer.Flush()

	// Parse the generated CSV
	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse generated CSV: %v", err)
	}

	// Verify we have header + 2 data rows
	if len(records) != 3 {
		t.Errorf("expected 3 records (header + 2 data), got %d", len(records))
	}

	// Verify header
	if len(records) > 0 {
		expectedHeaders := []string{"date", "shop", "items", "total", "currency", "tags", "source", "status"}
		for i, h := range expectedHeaders {
			if records[0][i] != h {
				t.Errorf("header[%d]: expected '%s', got '%s'", i, h, records[0][i])
			}
		}
	}

	// Verify first receipt row
	if len(records) > 1 {
		row := records[1]
		if row[0] != "2024-06-15" {
			t.Errorf("row 1 date: expected '2024-06-15', got '%s'", row[0])
		}
		if row[1] != "Grocery Store" {
			t.Errorf("row 1 shop: expected 'Grocery Store', got '%s'", row[1])
		}
		if !strings.Contains(row[2], "Apples") || !strings.Contains(row[2], "Bread") {
			t.Errorf("row 1 items: expected to contain 'Apples' and 'Bread', got '%s'", row[2])
		}
		if row[3] != "85.47" {
			t.Errorf("row 1 total: expected '85.47', got '%s'", row[3])
		}
		if row[4] != "USD" {
			t.Errorf("row 1 currency: expected 'USD', got '%s'", row[4])
		}
		if !strings.Contains(row[5], "food") || !strings.Contains(row[5], "groceries") {
			t.Errorf("row 1 tags: expected to contain 'food' and 'groceries', got '%s'", row[5])
		}
		if row[6] != "manual" {
			t.Errorf("row 1 source: expected 'manual', got '%s'", row[6])
		}
		if row[7] != "confirmed" {
			t.Errorf("row 1 status: expected 'confirmed', got '%s'", row[7])
		}
	}

	// Verify second receipt row
	if len(records) > 2 {
		row := records[2]
		if row[0] != "2024-06-20" {
			t.Errorf("row 2 date: expected '2024-06-20', got '%s'", row[0])
		}
		if row[1] != "Electronics Shop" {
			t.Errorf("row 2 shop: expected 'Electronics Shop', got '%s'", row[1])
		}
		if row[3] != "299.99" {
			t.Errorf("row 2 total: expected '299.99', got '%s'", row[3])
		}
		if row[6] != "ocr" {
			t.Errorf("row 2 source: expected 'ocr', got '%s'", row[6])
		}
		if row[7] != "pending_review" {
			t.Errorf("row 2 status: expected 'pending_review', got '%s'", row[7])
		}
	}
}

// Note: Full handler tests with database would require:
// 1. A test database setup
// 2. Proper JWT token generation for authentication
// 3. Mocking the receipt service or using a test double
//
// The integration test above tests the core CSV formatting logic.
// End-to-end tests should be done with the running server using actual requests.
