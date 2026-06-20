package models

import "time"

type DepartmentBudget struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DepartmentID   string    `gorm:"size:50;uniqueIndex" json:"department_id"`
	DepartmentName string    `gorm:"size:100" json:"department_name"`
	TotalBudget    float64   `json:"total_budget"`
	UsedBudget     float64   `json:"used_budget"`
	RemainingBudget float64  `json:"remaining_budget"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ExpenseItem struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	ExpenseReportID uint  `json:"-"`
	ItemType      string  `gorm:"size:50" json:"item_type"`
	Description   string  `gorm:"size:255" json:"description"`
	Amount        float64 `json:"amount"`
}

type ExpenseReport struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	ReportNo       string        `gorm:"size:50;uniqueIndex" json:"report_no"`
	IdempotencyKey *string       `gorm:"size:64;uniqueIndex" json:"idempotency_key,omitempty"`
	ContentHash    string        `gorm:"size:64;uniqueIndex" json:"content_hash,omitempty"`
	EmployeeID     string        `gorm:"size:50" json:"employee_id"`
	EmployeeName   string        `gorm:"size:100" json:"employee_name"`
	DepartmentID   string        `gorm:"size:50;index" json:"department_id"`
	DepartmentName string        `gorm:"size:100" json:"department_name"`
	TravelPurpose  string        `gorm:"size:255" json:"travel_purpose"`
	StartDate      time.Time     `json:"start_date"`
	EndDate        time.Time     `json:"end_date"`
	TotalAmount    float64       `json:"total_amount"`
	Status         string        `gorm:"size:20;default:'pending'" json:"status"`
	Items          []ExpenseItem `gorm:"foreignKey:ExpenseReportID" json:"items"`
	RejectionReason string       `gorm:"size:255" json:"rejection_reason,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type ExpenseReportRequest struct {
	EmployeeID    string    `json:"employee_id" binding:"required"`
	EmployeeName  string    `json:"employee_name" binding:"required"`
	DepartmentID  string    `json:"department_id" binding:"required"`
	TravelPurpose string    `json:"travel_purpose" binding:"required"`
	StartDate     string    `json:"start_date" binding:"required"`
	EndDate       string    `json:"end_date" binding:"required"`
	Items         []ExpenseItemRequest `json:"items" binding:"required,min=1,dive"`
}

type ExpenseItemRequest struct {
	ItemType    string  `json:"item_type" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
}

type BudgetCheckResult struct {
	Pass            bool    `json:"pass"`
	TotalAmount     float64 `json:"total_amount"`
	RemainingBudget float64 `json:"remaining_budget"`
	Message         string  `json:"message"`
}
