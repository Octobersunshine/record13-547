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

func TestGenerateContentHash_Consistent(t *testing.T) {
	req1 := &models.ExpenseReportRequest{
		EmployeeID:    "EMP001",
		DepartmentID:  "DEPT001",
		TravelPurpose: "测试",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
			{ItemType: "住宿", Description: "酒店", Amount: 500},
		},
	}

	req2 := &models.ExpenseReportRequest{
		EmployeeID:    "EMP001",
		DepartmentID:  "DEPT001",
		TravelPurpose: "测试",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "住宿", Description: "酒店", Amount: 500},
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	hash1 := GenerateContentHash(req1)
	hash2 := GenerateContentHash(req2)

	assert.Equal(t, hash1, hash2, "相同内容不同顺序应生成相同哈希")
}

func TestGenerateContentHash_Different(t *testing.T) {
	req1 := &models.ExpenseReportRequest{
		EmployeeID:    "EMP001",
		DepartmentID:  "DEPT001",
		TravelPurpose: "测试1",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	req2 := &models.ExpenseReportRequest{
		EmployeeID:    "EMP001",
		DepartmentID:  "DEPT001",
		TravelPurpose: "测试2",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	hash1 := GenerateContentHash(req1)
	hash2 := GenerateContentHash(req2)

	assert.NotEqual(t, hash1, hash2, "不同内容应生成不同哈希")
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

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsDuplicate)
	assert.Equal(t, "approved", result.Report.Status)
	assert.Equal(t, 4000.0, result.Report.TotalAmount)
	assert.Equal(t, 3, len(result.Report.Items))
	assert.NotEmpty(t, result.Report.ReportNo)
	assert.NotEmpty(t, result.Report.ContentHash)

	budget, _ := database.GetDepartmentBudget("DEPT001")
	assert.Equal(t, 4000.0, budget.UsedBudget)
	assert.Equal(t, 46000.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_DuplicateByContentHash(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP002",
		EmployeeName:  "李四",
		DepartmentID:  "DEPT002",
		TravelPurpose: "客户拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "打车", Amount: 300},
			{ItemType: "餐饮", Description: "午餐", Amount: 200},
		},
	}

	result1, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)
	assert.Equal(t, "approved", result1.Report.Status)

	originalBudget, _ := database.GetDepartmentBudget("DEPT002")
	originalUsed := originalBudget.UsedBudget

	req.Items = []models.ExpenseItemRequest{
		{ItemType: "餐饮", Description: "午餐", Amount: 200},
		{ItemType: "交通", Description: "打车", Amount: 300},
	}

	result2, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "content_hash", result2.DuplicateBy)
	assert.Equal(t, result1.Report.ID, result2.Report.ID)

	finalBudget, _ := database.GetDepartmentBudget("DEPT002")
	assert.Equal(t, originalUsed, finalBudget.UsedBudget, "重复提交不应重复扣减预算")
}

func TestSubmitExpenseReport_DuplicateByIdempotencyKey(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP003",
		EmployeeName:  "王五",
		DepartmentID:  "DEPT003",
		TravelPurpose: "市场调研",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-22",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "高铁", Amount: 1500},
		},
	}

	idempotencyKey := "unique-key-12345"

	result1, err := SubmitExpenseReport(req, idempotencyKey)
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)
	assert.Equal(t, "approved", result1.Report.Status)

	originalBudget, _ := database.GetDepartmentBudget("DEPT003")
	originalUsed := originalBudget.UsedBudget

	req.Items = []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "高铁", Amount: 2000},
	}

	result2, err := SubmitExpenseReport(req, idempotencyKey)
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "idempotency_key", result2.DuplicateBy)
	assert.Equal(t, result1.Report.ID, result2.Report.ID)
	assert.Equal(t, 1500.0, result2.Report.TotalAmount, "应返回原始金额，不是新金额")

	finalBudget, _ := database.GetDepartmentBudget("DEPT003")
	assert.Equal(t, originalUsed, finalBudget.UsedBudget, "重复提交不应重复扣减预算")
}

func TestSubmitExpenseReport_MultipleSubmissionsSameContent(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP004",
		EmployeeName:  "赵六",
		DepartmentID:  "DEPT004",
		TravelPurpose: "财务审计",
		StartDate:     "2026-06-25",
		EndDate:       "2026-06-26",
		Items: []models.ExpenseItemRequest{
			{ItemType: "住宿", Description: "酒店", Amount: 800},
		},
	}

	result1, err := SubmitExpenseReport(req, "key-1")
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)

	result2, err := SubmitExpenseReport(req, "key-2")
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "content_hash", result2.DuplicateBy)
	assert.Equal(t, result1.Report.ID, result2.Report.ID)

	budget, _ := database.GetDepartmentBudget("DEPT004")
	assert.Equal(t, 800.0, budget.UsedBudget, "只应扣减一次预算")
}

func TestSubmitExpenseReport_BudgetExceeded(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP005",
		EmployeeName:  "钱七",
		DepartmentID:  "DEPT005",
		TravelPurpose: "招聘面试",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "高铁", Amount: 10000},
			{ItemType: "住宿", Description: "酒店", Amount: 8000},
		},
	}

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsDuplicate)
	assert.Equal(t, "rejected", result.Report.Status)
	assert.Equal(t, 18000.0, result.Report.TotalAmount)
	assert.NotEmpty(t, result.Report.RejectionReason)

	budget, _ := database.GetDepartmentBudget("DEPT005")
	assert.Equal(t, 0.0, budget.UsedBudget)
	assert.Equal(t, 15000.0, budget.RemainingBudget)

	result2, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, result.Report.ID, result2.Report.ID)
}

func TestSubmitExpenseReport_InvalidDate(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP006",
		EmployeeName:  "孙八",
		DepartmentID:  "DEPT001",
		TravelPurpose: "出差",
		StartDate:     "2026-06-25",
		EndDate:       "2026-06-20",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	_, err := SubmitExpenseReport(req, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "结束日期不能早于开始日期")
}

func TestSubmitExpenseReport_InvalidDateFormat(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP007",
		EmployeeName:  "周九",
		DepartmentID:  "DEPT001",
		TravelPurpose: "出差",
		StartDate:     "2026/06/20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 1000},
		},
	}

	_, err := SubmitExpenseReport(req, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "开始日期格式错误")
}

func TestGetExpenseReport(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP008",
		EmployeeName:  "吴十",
		DepartmentID:  "DEPT002",
		TravelPurpose: "客户拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "打车", Amount: 300},
		},
	}

	created, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)

	fetched, err := GetExpenseReport(created.Report.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.Report.ID, fetched.ID)
	assert.Equal(t, created.Report.ReportNo, fetched.ReportNo)
	assert.Equal(t, 1, len(fetched.Items))
	assert.NotEmpty(t, fetched.ContentHash)
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
