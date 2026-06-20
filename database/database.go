package database

import (
	"log"
	"travel-expense/models"

	_ "modernc.org/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	InitWithDB("expense.db")
}

func InitWithDB(dbName string) {
	var err error
	DB, err = gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbName,
	}, &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = DB.AutoMigrate(&models.DepartmentBudget{}, &models.ExpenseReport{}, &models.ExpenseItem{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	seedData()
}

func seedData() {
	var count int64
	DB.Model(&models.DepartmentBudget{}).Count(&count)
	if count > 0 {
		return
	}

	budgets := []models.DepartmentBudget{
		{DepartmentID: "DEPT001", DepartmentName: "技术部", TotalBudget: 50000, UsedBudget: 0, RemainingBudget: 50000},
		{DepartmentID: "DEPT002", DepartmentName: "销售部", TotalBudget: 80000, UsedBudget: 0, RemainingBudget: 80000},
		{DepartmentID: "DEPT003", DepartmentName: "市场部", TotalBudget: 30000, UsedBudget: 0, RemainingBudget: 30000},
		{DepartmentID: "DEPT004", DepartmentName: "财务部", TotalBudget: 20000, UsedBudget: 0, RemainingBudget: 20000},
		{DepartmentID: "DEPT005", DepartmentName: "人事部", TotalBudget: 15000, UsedBudget: 0, RemainingBudget: 15000},
	}

	for _, budget := range budgets {
		DB.Create(&budget)
	}

	log.Println("seed data created successfully")
}

func GetDepartmentBudget(departmentID string) (*models.DepartmentBudget, error) {
	var budget models.DepartmentBudget
	result := DB.Where("department_id = ?", departmentID).First(&budget)
	if result.Error != nil {
		return nil, result.Error
	}
	return &budget, nil
}

func CreateExpenseReport(report *models.ExpenseReport) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var budget models.DepartmentBudget
		if err := tx.Where("department_id = ?", report.DepartmentID).First(&budget).Error; err != nil {
			return err
		}

		if report.Status == "approved" {
			budget.UsedBudget += report.TotalAmount
			budget.RemainingBudget = budget.TotalBudget - budget.UsedBudget
			if err := tx.Save(&budget).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(report).Error; err != nil {
			return err
		}

		return nil
	})
}

func GetExpenseReportByID(id uint) (*models.ExpenseReport, error) {
	var report models.ExpenseReport
	result := DB.Preload("Items").First(&report, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &report, nil
}

func GetExpenseReports(departmentID, status string) ([]models.ExpenseReport, error) {
	var reports []models.ExpenseReport
	query := DB.Preload("Items")

	if departmentID != "" {
		query = query.Where("department_id = ?", departmentID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	result := query.Order("created_at desc").Find(&reports)
	if result.Error != nil {
		return nil, result.Error
	}
	return reports, nil
}

func GetAllDepartments() ([]models.DepartmentBudget, error) {
	var budgets []models.DepartmentBudget
	result := DB.Find(&budgets)
	if result.Error != nil {
		return nil, result.Error
	}
	return budgets, nil
}

func UpdateDepartmentBudget(budget *models.DepartmentBudget) error {
	return DB.Save(budget).Error
}

func GetExpenseReportByIdempotencyKey(key string) (*models.ExpenseReport, error) {
	var report models.ExpenseReport
	result := DB.Preload("Items").Where("idempotency_key = ?", key).First(&report)
	if result.Error != nil {
		return nil, result.Error
	}
	return &report, nil
}

func GetExpenseReportByContentHash(hash string) (*models.ExpenseReport, error) {
	var report models.ExpenseReport
	result := DB.Preload("Items").Where("content_hash = ?", hash).First(&report)
	if result.Error != nil {
		return nil, result.Error
	}
	return &report, nil
}
