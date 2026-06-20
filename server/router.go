package server

import (
	"travel-expense/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(CORS())

	handler := handlers.NewExpenseHandler()

	api := r.Group("/api/v1")
	{
		expense := api.Group("/expense")
		{
			expense.GET("/budget/check", handler.CheckBudget)
			expense.POST("/submit", handler.SubmitExpense)
			expense.GET("/:id", handler.GetExpense)
			expense.GET("/list", handler.ListExpenses)
		}

		department := api.Group("/department")
		{
			department.GET("/list", handler.ListDepartments)
			department.GET("/budget/:department_id", handler.GetDepartmentBudget)
		}
	}

	return r
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
