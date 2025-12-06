package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// ExpenseItem represents a single expense line item
type ExpenseItem struct {
	ID          string    `json:"id"`
	EmployeeID  string    `json:"employee_id"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // approved, pending, rejected
	ReceiptURL  string    `json:"receipt_url"`
	ApprovedBy  string    `json:"approved_by"`
}

// TeamMember represents an employee
type TeamMember struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Email      string `json:"email"`
}

// GetTeamMembersTool returns team members for a department
type GetTeamMembersTool struct{}

func (t GetTeamMembersTool) Name() string {
	return "get_team_members"
}

func (t GetTeamMembersTool) Description() string {
	return "Returns a list of team members for a given department. Input should be the department name (e.g., 'engineering', 'sales', 'marketing')."
}

func (t GetTeamMembersTool) Call(ctx context.Context, input string) (string, error) {
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputData); err != nil {
		// Try direct string input
		inputData = map[string]interface{}{"department": input}
	}

	department, ok := inputData["department"].(string)
	if !ok {
		return "", fmt.Errorf("department is required")
	}

	// Mock data
	members := []TeamMember{}

	switch department {
	case "engineering":
		members = []TeamMember{
			{ID: "E001", Name: "Alice Johnson", Department: "engineering", Email: "alice@example.com"},
			{ID: "E002", Name: "Bob Smith", Department: "engineering", Email: "bob@example.com"},
			{ID: "E003", Name: "Charlie Davis", Department: "engineering", Email: "charlie@example.com"},
			{ID: "E004", Name: "Diana Lee", Department: "engineering", Email: "diana@example.com"},
		}
	case "sales":
		members = []TeamMember{
			{ID: "S001", Name: "Eve Wilson", Department: "sales", Email: "eve@example.com"},
			{ID: "S002", Name: "Frank Brown", Department: "sales", Email: "frank@example.com"},
		}
	case "marketing":
		members = []TeamMember{
			{ID: "M001", Name: "Grace Taylor", Department: "marketing", Email: "grace@example.com"},
		}
	default:
		return "", fmt.Errorf("unknown department: %s", department)
	}

	result, _ := json.MarshalIndent(members, "", "  ")
	return string(result), nil
}

// GetExpensesTool returns expenses for an employee in a quarter
type GetExpensesTool struct{}

func (t GetExpensesTool) Name() string {
	return "get_expenses"
}

func (t GetExpensesTool) Description() string {
	return "Returns all expense line items for a given employee in a specific quarter. Input should be JSON with 'employee_id' and 'quarter' (e.g., 'Q1', 'Q2', 'Q3', 'Q4')."
}

func (t GetExpensesTool) Call(ctx context.Context, input string) (string, error) {
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputData); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	employeeID, ok := inputData["employee_id"].(string)
	if !ok {
		return "", fmt.Errorf("employee_id is required")
	}

	quarter, ok := inputData["quarter"].(string)
	if !ok {
		return "", fmt.Errorf("quarter is required")
	}

	// Generate mock expenses
	expenses := generateMockExpenses(employeeID, quarter)

	result, _ := json.MarshalIndent(expenses, "", "  ")
	return string(result), nil
}

// GetCustomBudgetTool returns custom budget for an employee
type GetCustomBudgetTool struct{}

func (t GetCustomBudgetTool) Name() string {
	return "get_custom_budget"
}

func (t GetCustomBudgetTool) Description() string {
	return "Get the custom quarterly travel budget for a specific employee. Input should be JSON with 'user_id'. Returns the custom budget amount if one exists, otherwise returns null."
}

func (t GetCustomBudgetTool) Call(ctx context.Context, input string) (string, error) {
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputData); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	userID, ok := inputData["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id is required")
	}

	// Mock custom budgets (some employees have higher limits)
	customBudgets := map[string]float64{
		"E001": 7500.0,  // Alice has higher budget
		"E003": 10000.0, // Charlie has higher budget
	}

	if budget, exists := customBudgets[userID]; exists {
		result := map[string]interface{}{
			"user_id": userID,
			"budget":  budget,
		}
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return string(resultJSON), nil
	}

	return `{"user_id": "` + userID + `", "budget": null}`, nil
}

// generateMockExpenses generates mock expense data
func generateMockExpenses(employeeID, quarter string) []ExpenseItem {
	rand.Seed(time.Now().UnixNano() + int64(len(employeeID)))

	categories := []string{"travel", "meals", "accommodation", "transportation", "office supplies"}
	statuses := []string{"approved", "approved", "approved", "pending"}

	numExpenses := 15 + rand.Intn(20) // 15-35 expenses

	expenses := make([]ExpenseItem, numExpenses)
	totalAmount := 0.0

	// Some employees spend more
	multiplier := 1.0
	if employeeID == "E002" || employeeID == "E004" {
		multiplier = 1.5 // Bob and Diana spend more
	}

	for i := 0; i < numExpenses; i++ {
		category := categories[rand.Intn(len(categories))]
		amount := (50.0 + rand.Float64()*500.0) * multiplier

		// Travel expenses are larger
		if category == "travel" || category == "accommodation" {
			amount *= 2
		}

		expenses[i] = ExpenseItem{
			ID:          fmt.Sprintf("EXP-%s-%s-%03d", employeeID, quarter, i+1),
			EmployeeID:  employeeID,
			Category:    category,
			Amount:      amount,
			Date:        time.Now().AddDate(0, -rand.Intn(3), -rand.Intn(30)),
			Description: fmt.Sprintf("%s expense for %s", category, employeeID),
			Status:      statuses[rand.Intn(len(statuses))],
			ReceiptURL:  fmt.Sprintf("https://receipts.example.com/%s-%d", employeeID, i+1),
			ApprovedBy:  "manager@example.com",
		}
		totalAmount += amount
	}

	return expenses
}
