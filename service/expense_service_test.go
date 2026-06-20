package service

import (
	"os"
	"testing"
	"travel-expense/database"
	"travel-expense/models"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Remove("expense_test.db")
	database.DB = nil

	database.InitWithDB("expense_test.db")

	code := m.Run()

	os.Remove("expense_test.db")

	os.Exit(code)
}

func TestCheckBudget_Sufficient(t *testing.T) {
	result, err := CheckBudget("DEPT001", 10000)
	assert.NoError(t, err)
	assert.True(t, result.Pass)
	assert.Equal(t, 10000.0, result.TotalAmount)
	assert.Equal(t, 50000.0, result.RemainingBudget)
}

func TestCheckBudget_Insufficient(t *testing.T) {
	result, err := CheckBudget("DEPT001", 60000)
	assert.NoError(t, err)
	assert.False(t, result.Pass)
	assert.Equal(t, 60000.0, result.TotalAmount)
	assert.Equal(t, 50000.0, result.RemainingBudget)
}

func TestCheckBudget_DepartmentNotFound(t *testing.T) {
	_, err := CheckBudget("INVALID", 1000)
	assert.Error(t, err)
}

func TestSubmitExpenseReport_Success(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP001",
		EmployeeName:  "张三",
		DepartmentID:  "DEPT001",
		TravelPurpose: "参加技术会议",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-22",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 2000},
			{ItemType: "住宿", Description: "酒店", Amount: 1500},
			{ItemType: "餐饮", Description: "餐费", Amount: 500},
		},
	}

	report, err := SubmitExpenseReport(req)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "approved", report.Status)
	assert.Equal(t, 4000.0, report.TotalAmount)
	assert.Equal(t, 3, len(report.Items))
	assert.NotEmpty(t, report.ReportNo)

	budget, _ := database.GetDepartmentBudget("DEPT001")
	assert.Equal(t, 4000.0, budget.UsedBudget)
	assert.Equal(t, 46000.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_BudgetExceeded(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP002",
		EmployeeName:  "李四",
		DepartmentID:  "DEPT005",
		TravelPurpose: "招聘面试",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "高铁", Amount: 10000},
			{ItemType: "住宿", Description: "酒店", Amount: 8000},
		},
	}

	report, err := SubmitExpenseReport(req)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "rejected", report.Status)
	assert.Equal(t, 18000.0, report.TotalAmount)
	assert.NotEmpty(t, report.RejectionReason)

	budget, _ := database.GetDepartmentBudget("DEPT005")
	assert.Equal(t, 0.0, budget.UsedBudget)
	assert.Equal(t, 15000.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_InvalidDate(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP003",
		EmployeeName:  "王五",
		DepartmentID:  "DEPT001",
		TravelPurpose: "出差",
		StartDate:     "2026-06-25",
		EndDate:       "2026-06-20",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	_, err := SubmitExpenseReport(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "结束日期不能早于开始日期")
}

func TestSubmitExpenseReport_InvalidDateFormat(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP004",
		EmployeeName:  "赵六",
		DepartmentID:  "DEPT001",
		TravelPurpose: "出差",
		StartDate:     "2026/06/20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	_, err := SubmitExpenseReport(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "开始日期格式错误")
}

func TestGetExpenseReport(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP005",
		EmployeeName:  "钱七",
		DepartmentID:  "DEPT002",
		TravelPurpose: "客户拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "打车", Amount: 300},
		},
	}

	created, err := SubmitExpenseReport(req)
	assert.NoError(t, err)

	fetched, err := GetExpenseReport(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, created.ReportNo, fetched.ReportNo)
	assert.Equal(t, 1, len(fetched.Items))
}

func TestListExpenseReports(t *testing.T) {
	reports, err := ListExpenseReports("", "")
	assert.NoError(t, err)
	assert.Greater(t, len(reports), 0)
}

func TestListExpenseReports_ByDepartment(t *testing.T) {
	reports, err := ListExpenseReports("DEPT001", "")
	assert.NoError(t, err)
	for _, r := range reports {
		assert.Equal(t, "DEPT001", r.DepartmentID)
	}
}

func TestListExpenseReports_ByStatus(t *testing.T) {
	reports, err := ListExpenseReports("", "approved")
	assert.NoError(t, err)
	for _, r := range reports {
		assert.Equal(t, "approved", r.Status)
	}
}

func TestListDepartments(t *testing.T) {
	departments, err := ListDepartments()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(departments))
}

func TestGetDepartmentBudgetInfo(t *testing.T) {
	budget, err := GetDepartmentBudgetInfo("DEPT001")
	assert.NoError(t, err)
	assert.Equal(t, "DEPT001", budget.DepartmentID)
	assert.Equal(t, "技术部", budget.DepartmentName)
	assert.Equal(t, 50000.0, budget.TotalBudget)
}
