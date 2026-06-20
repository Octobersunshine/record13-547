package service

import (
	"errors"
	"fmt"
	"time"
	"travel-expense/database"
	"travel-expense/models"
)

func CheckBudget(departmentID string, totalAmount float64) (*models.BudgetCheckResult, error) {
	budget, err := database.GetDepartmentBudget(departmentID)
	if err != nil {
		return nil, fmt.Errorf("部门预算不存在: %w", err)
	}

	result := &models.BudgetCheckResult{
		TotalAmount:     totalAmount,
		RemainingBudget: budget.RemainingBudget,
	}

	if totalAmount > budget.RemainingBudget {
		result.Pass = false
		result.Message = fmt.Sprintf("预算不足！报销金额: %.2f 元，剩余预算: %.2f 元，超出 %.2f 元",
			totalAmount, budget.RemainingBudget, totalAmount-budget.RemainingBudget)
		return result, nil
	}

	result.Pass = true
	result.Message = fmt.Sprintf("预算充足。报销金额: %.2f 元，剩余预算: %.2f 元，提交后剩余: %.2f 元",
		totalAmount, budget.RemainingBudget, budget.RemainingBudget-totalAmount)
	return result, nil
}

func SubmitExpenseReport(req *models.ExpenseReportRequest) (*models.ExpenseReport, error) {
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.Amount
	}

	budgetCheck, err := CheckBudget(req.DepartmentID, totalAmount)
	if err != nil {
		return nil, err
	}

	budget, err := database.GetDepartmentBudget(req.DepartmentID)
	if err != nil {
		return nil, fmt.Errorf("获取部门预算失败: %w", err)
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}

	if endDate.Before(startDate) {
		return nil, errors.New("结束日期不能早于开始日期")
	}

	reportNo := fmt.Sprintf("EXP%s%06d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)

	status := "approved"
	rejectionReason := ""

	if !budgetCheck.Pass {
		status = "rejected"
		rejectionReason = budgetCheck.Message
	}

	report := &models.ExpenseReport{
		ReportNo:        reportNo,
		EmployeeID:      req.EmployeeID,
		EmployeeName:    req.EmployeeName,
		DepartmentID:    req.DepartmentID,
		DepartmentName:  budget.DepartmentName,
		TravelPurpose:   req.TravelPurpose,
		StartDate:       startDate,
		EndDate:         endDate,
		TotalAmount:     totalAmount,
		Status:          status,
		RejectionReason: rejectionReason,
		Items:           make([]models.ExpenseItem, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		report.Items = append(report.Items, models.ExpenseItem{
			ItemType:    item.ItemType,
			Description: item.Description,
			Amount:      item.Amount,
		})
	}

	err = database.CreateExpenseReport(report)
	if err != nil {
		return nil, fmt.Errorf("创建报销单失败: %w", err)
	}

	return report, nil
}

func GetExpenseReport(id uint) (*models.ExpenseReport, error) {
	return database.GetExpenseReportByID(id)
}

func ListExpenseReports(departmentID, status string) ([]models.ExpenseReport, error) {
	return database.GetExpenseReports(departmentID, status)
}

func ListDepartments() ([]models.DepartmentBudget, error) {
	return database.GetAllDepartments()
}

func GetDepartmentBudgetInfo(departmentID string) (*models.DepartmentBudget, error) {
	return database.GetDepartmentBudget(departmentID)
}
