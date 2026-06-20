package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"travel-expense/database"
	"travel-expense/models"
)

type SubmitResult struct {
	Report      *models.ExpenseReport
	IsDuplicate bool
	DuplicateBy string
}

func GenerateContentHash(req *models.ExpenseReportRequest) string {
	type itemForHash struct {
		ItemType    string
		Description string
		Amount      float64
	}

	items := make([]itemForHash, len(req.Items))
	for i, item := range req.Items {
		items[i] = itemForHash{
			ItemType:    item.ItemType,
			Description: item.Description,
			Amount:      item.Amount,
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ItemType != items[j].ItemType {
			return items[i].ItemType < items[j].ItemType
		}
		if items[i].Description != items[j].Description {
			return items[i].Description < items[j].Description
		}
		return items[i].Amount < items[j].Amount
	})

	var content strings.Builder
	content.WriteString(req.EmployeeID)
	content.WriteString("|")
	content.WriteString(req.DepartmentID)
	content.WriteString("|")
	content.WriteString(req.TravelPurpose)
	content.WriteString("|")
	content.WriteString(req.StartDate)
	content.WriteString("|")
	content.WriteString(req.EndDate)
	content.WriteString("|")

	for _, item := range items {
		content.WriteString(fmt.Sprintf("%s:%s:%.2f;", item.ItemType, item.Description, item.Amount))
	}

	hash := sha256.Sum256([]byte(content.String()))
	return hex.EncodeToString(hash[:])
}

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

func SubmitExpenseReport(req *models.ExpenseReportRequest, idempotencyKey string) (*SubmitResult, error) {
	contentHash := GenerateContentHash(req)

	if idempotencyKey != "" {
		existing, err := database.GetExpenseReportByIdempotencyKey(idempotencyKey)
		if err == nil && existing != nil {
			return &SubmitResult{
				Report:      existing,
				IsDuplicate: true,
				DuplicateBy: "idempotency_key",
			}, nil
		}
	}

	existingByHash, err := database.GetExpenseReportByContentHash(contentHash)
	if err == nil && existingByHash != nil {
		return &SubmitResult{
			Report:      existingByHash,
			IsDuplicate: true,
			DuplicateBy: "content_hash",
		}, nil
	}

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

	var idemKey *string
	if idempotencyKey != "" {
		idemKey = &idempotencyKey
	}

	report := &models.ExpenseReport{
		ReportNo:        reportNo,
		IdempotencyKey:  idemKey,
		ContentHash:     contentHash,
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
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if idempotencyKey != "" {
				existing, _ := database.GetExpenseReportByIdempotencyKey(idempotencyKey)
				if existing != nil {
					return &SubmitResult{
						Report:      existing,
						IsDuplicate: true,
						DuplicateBy: "idempotency_key",
					}, nil
				}
			}
			existingByHash, _ := database.GetExpenseReportByContentHash(contentHash)
			if existingByHash != nil {
				return &SubmitResult{
					Report:      existingByHash,
					IsDuplicate: true,
					DuplicateBy: "content_hash",
				}, nil
			}
		}
		return nil, fmt.Errorf("创建报销单失败: %w", err)
	}

	return &SubmitResult{
		Report:      report,
		IsDuplicate: false,
		DuplicateBy: "",
	}, nil
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
