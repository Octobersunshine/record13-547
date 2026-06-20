package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"travel-expense/models"
	"travel-expense/service"

	"github.com/gin-gonic/gin"
)

type ExpenseHandler struct{}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{}
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *ExpenseHandler) CheckBudget(c *gin.Context) {
	departmentID := c.Query("department_id")
	amountStr := c.Query("amount")

	if departmentID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "部门ID不能为空",
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "金额格式错误",
		})
		return
	}

	if amount <= 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "金额必须大于0",
		})
		return
	}

	result, err := service.CheckBudget(departmentID, amount)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    result,
	})
}

type SubmitResponse struct {
	IsDuplicate   bool                   `json:"is_duplicate"`
	DuplicateBy   string                 `json:"duplicate_by,omitempty"`
	ExpenseReport *models.ExpenseReport  `json:"expense_report"`
}

func (h *ExpenseHandler) SubmitExpense(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")

	var req models.ExpenseReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	result, err := service.SubmitExpenseReport(&req, idempotencyKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	statusMsg := "报销单提交成功"
	if result.Report.Status == "partial_approved" {
		statusMsg = fmt.Sprintf("报销单部分通过：超标部分转个人结算（公司: %.2f 元，个人: %.2f 元）",
			result.Report.CompanyAmount, result.Report.PersonalAmount)
	}

	if result.IsDuplicate {
		duplicateType := "内容"
		if result.DuplicateBy == "idempotency_key" {
			duplicateType = "幂等键"
		}
		statusMsg = fmt.Sprintf("重复提交检测：该报销单已存在（%s匹配），返回原始记录", duplicateType)
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: statusMsg,
		Data: SubmitResponse{
			IsDuplicate:   result.IsDuplicate,
			DuplicateBy:   result.DuplicateBy,
			ExpenseReport: result.Report,
		},
	})
}

func (h *ExpenseHandler) GetExpense(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "ID格式错误",
		})
		return
	}

	report, err := service.GetExpenseReport(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "报销单不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    report,
	})
}

func (h *ExpenseHandler) ListExpenses(c *gin.Context) {
	departmentID := c.Query("department_id")
	status := c.Query("status")

	reports, err := service.ListExpenseReports(departmentID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    reports,
	})
}

func (h *ExpenseHandler) ListDepartments(c *gin.Context) {
	departments, err := service.ListDepartments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    departments,
	})
}

func (h *ExpenseHandler) GetDepartmentBudget(c *gin.Context) {
	departmentID := c.Param("department_id")

	budget, err := service.GetDepartmentBudgetInfo(departmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "部门预算不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    budget,
	})
}
