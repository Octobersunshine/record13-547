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

func TestSplitPayment_AllCompany(t *testing.T) {
	items := []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "机票", Amount: 2000},
		{ItemType: "住宿", Description: "酒店", Amount: 1500},
	}
	split := SplitPayment(items, 50000)

	assert.Equal(t, 3500.0, split.CompanyAmount)
	assert.Equal(t, 0.0, split.PersonalAmount)
	assert.Equal(t, "company", split.Items[0].PayBy)
	assert.Equal(t, "company", split.Items[1].PayBy)
}

func TestSplitPayment_AllPersonal(t *testing.T) {
	items := []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "机票", Amount: 2000},
		{ItemType: "住宿", Description: "酒店", Amount: 1500},
	}
	split := SplitPayment(items, 0)

	assert.Equal(t, 0.0, split.CompanyAmount)
	assert.Equal(t, 3500.0, split.PersonalAmount)
	assert.Equal(t, "personal", split.Items[0].PayBy)
	assert.Equal(t, "personal", split.Items[1].PayBy)
}

func TestSplitPayment_PartialSplit(t *testing.T) {
	items := []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "机票", Amount: 5000},
		{ItemType: "住宿", Description: "酒店", Amount: 3000},
		{ItemType: "餐饮", Description: "餐费", Amount: 2000},
	}
	split := SplitPayment(items, 6000)

	assert.Equal(t, 6000.0, split.CompanyAmount)
	assert.Equal(t, 4000.0, split.PersonalAmount)

	assert.Equal(t, "company", split.Items[0].PayBy)
	assert.Equal(t, 5000.0, split.Items[0].CompanyAmount)
	assert.Equal(t, 0.0, split.Items[0].PersonalAmount)

	assert.Equal(t, "split", split.Items[1].PayBy)
	assert.Equal(t, 1000.0, split.Items[1].CompanyAmount)
	assert.Equal(t, 2000.0, split.Items[1].PersonalAmount)

	assert.Equal(t, "personal", split.Items[2].PayBy)
	assert.Equal(t, 0.0, split.Items[2].CompanyAmount)
	assert.Equal(t, 2000.0, split.Items[2].PersonalAmount)
}

func TestSplitPayment_SingleItemExceeds(t *testing.T) {
	items := []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "机票", Amount: 10000},
	}
	split := SplitPayment(items, 3000)

	assert.Equal(t, 3000.0, split.CompanyAmount)
	assert.Equal(t, 7000.0, split.PersonalAmount)
	assert.Equal(t, "split", split.Items[0].PayBy)
	assert.Equal(t, 3000.0, split.Items[0].CompanyAmount)
	assert.Equal(t, 7000.0, split.Items[0].PersonalAmount)
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
	assert.Equal(t, 4000.0, result.Report.CompanyAmount)
	assert.Equal(t, 0.0, result.Report.PersonalAmount)
	assert.Equal(t, 3, len(result.Report.Items))
	assert.NotEmpty(t, result.Report.ReportNo)
	assert.NotEmpty(t, result.Report.ContentHash)

	for _, item := range result.Report.Items {
		assert.Equal(t, "company", item.PayBy)
		assert.Equal(t, item.Amount, item.CompanyAmount)
		assert.Equal(t, 0.0, item.PersonalAmount)
	}

	budget, _ := database.GetDepartmentBudget("DEPT001")
	assert.Equal(t, 4000.0, budget.UsedBudget)
	assert.Equal(t, 46000.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_PartialApproved(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP002",
		EmployeeName:  "李四",
		DepartmentID:  "DEPT005",
		TravelPurpose: "招聘面试",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "高铁", Amount: 10000},
			{ItemType: "住宿", Description: "酒店", Amount: 5000},
			{ItemType: "餐饮", Description: "餐费", Amount: 3000},
		},
	}

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsDuplicate)
	assert.Equal(t, "partial_approved", result.Report.Status)
	assert.Equal(t, 18000.0, result.Report.TotalAmount)
	assert.Equal(t, 15000.0, result.Report.CompanyAmount)
	assert.Equal(t, 3000.0, result.Report.PersonalAmount)
	assert.NotEmpty(t, result.Report.RejectionReason)
	assert.Contains(t, result.Report.RejectionReason, "个人承担")

	assert.Equal(t, "company", result.Report.Items[0].PayBy)
	assert.Equal(t, 10000.0, result.Report.Items[0].CompanyAmount)
	assert.Equal(t, 0.0, result.Report.Items[0].PersonalAmount)

	assert.Equal(t, "company", result.Report.Items[1].PayBy)
	assert.Equal(t, 5000.0, result.Report.Items[1].CompanyAmount)
	assert.Equal(t, 0.0, result.Report.Items[1].PersonalAmount)

	assert.Equal(t, "personal", result.Report.Items[2].PayBy)
	assert.Equal(t, 0.0, result.Report.Items[2].CompanyAmount)
	assert.Equal(t, 3000.0, result.Report.Items[2].PersonalAmount)

	budget, _ := database.GetDepartmentBudget("DEPT005")
	assert.Equal(t, 15000.0, budget.UsedBudget, "只应扣减公司承担部分")
	assert.Equal(t, 0.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_PartialApproved_ItemSplit(t *testing.T) {
	budget, _ := database.GetDepartmentBudget("DEPT002")
	remaining := budget.RemainingBudget

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP003B",
		EmployeeName:  "陈三",
		DepartmentID:  "DEPT002",
		TravelPurpose: "跨城拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: remaining - 3000},
			{ItemType: "住宿", Description: "酒店", Amount: 5000},
			{ItemType: "餐饮", Description: "餐费", Amount: 1000},
		},
	}

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.Equal(t, "partial_approved", result.Report.Status)

	totalAmount := (remaining - 3000) + 5000 + 1000
	assert.Equal(t, totalAmount, result.Report.TotalAmount)
	assert.Equal(t, remaining, result.Report.CompanyAmount)
	assert.Equal(t, 3000.0, result.Report.PersonalAmount)

	assert.Equal(t, "company", result.Report.Items[0].PayBy)
	assert.Equal(t, remaining-3000, result.Report.Items[0].CompanyAmount)
	assert.Equal(t, 0.0, result.Report.Items[0].PersonalAmount)

	assert.Equal(t, "split", result.Report.Items[1].PayBy)
	assert.Equal(t, 3000.0, result.Report.Items[1].CompanyAmount)
	assert.Equal(t, 2000.0, result.Report.Items[1].PersonalAmount)

	assert.Equal(t, "personal", result.Report.Items[2].PayBy)
	assert.Equal(t, 0.0, result.Report.Items[2].CompanyAmount)
	assert.Equal(t, 1000.0, result.Report.Items[2].PersonalAmount)
}

func TestSubmitExpenseReport_PartialApproved_SingleItemSplit(t *testing.T) {
	budget, _ := database.GetDepartmentBudget("DEPT001")
	remaining := budget.RemainingBudget

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP003",
		EmployeeName:  "王五",
		DepartmentID:  "DEPT001",
		TravelPurpose: "出差审计",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: remaining + 5000},
		},
	}

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.Equal(t, "partial_approved", result.Report.Status)
	assert.Equal(t, remaining+5000, result.Report.TotalAmount)
	assert.Equal(t, remaining, result.Report.CompanyAmount)
	assert.Equal(t, 5000.0, result.Report.PersonalAmount)

	assert.Equal(t, "split", result.Report.Items[0].PayBy)
	assert.Equal(t, remaining, result.Report.Items[0].CompanyAmount)
	assert.Equal(t, 5000.0, result.Report.Items[0].PersonalAmount)

	finalBudget, _ := database.GetDepartmentBudget("DEPT001")
	assert.Equal(t, 50000.0, finalBudget.UsedBudget)
	assert.Equal(t, 0.0, finalBudget.RemainingBudget)
}

func TestSubmitExpenseReport_BudgetExactlyExhausted(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP004",
		EmployeeName:  "赵六",
		DepartmentID:  "DEPT003",
		TravelPurpose: "市场活动",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "机票", Amount: 30000},
		},
	}

	result, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.Equal(t, "approved", result.Report.Status)
	assert.Equal(t, 30000.0, result.Report.CompanyAmount)
	assert.Equal(t, 0.0, result.Report.PersonalAmount)

	budget, _ := database.GetDepartmentBudget("DEPT003")
	assert.Equal(t, 30000.0, budget.UsedBudget)
	assert.Equal(t, 0.0, budget.RemainingBudget)
}

func TestSubmitExpenseReport_DuplicateByContentHash(t *testing.T) {
	budget, _ := database.GetDepartmentBudget("DEPT002")
	remaining := budget.RemainingBudget
	safeAmount := remaining * 0.1

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP005",
		EmployeeName:  "钱七",
		DepartmentID:  "DEPT002",
		TravelPurpose: "客户拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "打车", Amount: safeAmount * 0.6},
			{ItemType: "餐饮", Description: "午餐", Amount: safeAmount * 0.4},
		},
	}

	result1, err := SubmitExpenseReport(req, "")
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)
	assert.Equal(t, "approved", result1.Report.Status)

	originalBudget, _ := database.GetDepartmentBudget("DEPT002")
	originalUsed := originalBudget.UsedBudget

	req.Items = []models.ExpenseItemRequest{
		{ItemType: "餐饮", Description: "午餐", Amount: safeAmount * 0.4},
		{ItemType: "交通", Description: "打车", Amount: safeAmount * 0.6},
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
	budget, _ := database.GetDepartmentBudget("DEPT002")
	remaining := budget.RemainingBudget
	safeAmount := remaining * 0.1

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP006",
		EmployeeName:  "孙八",
		DepartmentID:  "DEPT002",
		TravelPurpose: "市场调研",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-22",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "高铁", Amount: safeAmount},
		},
	}

	idempotencyKey := "unique-key-67890"

	result1, err := SubmitExpenseReport(req, idempotencyKey)
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)
	assert.Equal(t, "approved", result1.Report.Status)

	originalBudget, _ := database.GetDepartmentBudget("DEPT002")
	originalUsed := originalBudget.UsedBudget

	req.Items = []models.ExpenseItemRequest{
		{ItemType: "交通", Description: "高铁", Amount: safeAmount * 1.5},
	}

	result2, err := SubmitExpenseReport(req, idempotencyKey)
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "idempotency_key", result2.DuplicateBy)
	assert.Equal(t, result1.Report.ID, result2.Report.ID)
	assert.Equal(t, safeAmount, result2.Report.TotalAmount, "应返回原始金额，不是新金额")

	finalBudget, _ := database.GetDepartmentBudget("DEPT002")
	assert.Equal(t, originalUsed, finalBudget.UsedBudget, "重复提交不应重复扣减预算")
}

func TestSubmitExpenseReport_MultipleSubmissionsSameContent(t *testing.T) {
	budget, _ := database.GetDepartmentBudget("DEPT002")
	remaining := budget.RemainingBudget
	safeAmount := remaining * 0.1

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP007",
		EmployeeName:  "周九",
		DepartmentID:  "DEPT002",
		TravelPurpose: "财务审计",
		StartDate:     "2026-06-25",
		EndDate:       "2026-06-26",
		Items: []models.ExpenseItemRequest{
			{ItemType: "住宿", Description: "酒店", Amount: safeAmount},
		},
	}

	result1, err := SubmitExpenseReport(req, "key-a")
	assert.NoError(t, err)
	assert.False(t, result1.IsDuplicate)

	result2, err := SubmitExpenseReport(req, "key-b")
	assert.NoError(t, err)
	assert.True(t, result2.IsDuplicate)
	assert.Equal(t, "content_hash", result2.DuplicateBy)
	assert.Equal(t, result1.Report.ID, result2.Report.ID)
}

func TestSubmitExpenseReport_InvalidDate(t *testing.T) {
	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP008",
		EmployeeName:  "吴十",
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
		EmployeeID:    "EMP009",
		EmployeeName:  "郑十一",
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
	deptBudget, _ := database.GetDepartmentBudget("DEPT004")
	safeAmount := deptBudget.RemainingBudget * 0.1
	if safeAmount < 1 {
		safeAmount = 1
	}

	req := &models.ExpenseReportRequest{
		EmployeeID:    "EMP010",
		EmployeeName:  "冯十二",
		DepartmentID:  "DEPT004",
		TravelPurpose: "客户拜访",
		StartDate:     "2026-06-20",
		EndDate:       "2026-06-21",
		Items: []models.ExpenseItemRequest{
			{ItemType: "交通", Description: "打车", Amount: safeAmount},
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
