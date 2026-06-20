package main

import (
	"log"
	"travel-expense/database"
	"travel-expense/server"
)

func main() {
	database.Init()
	log.Println("Database initialized successfully")

	r := server.SetupRouter()

	log.Println("Server starting on :8080")
	log.Println("API Endpoints:")
	log.Println("  GET    /api/v1/expense/budget/check       - 预算检查")
	log.Println("  POST   /api/v1/expense/submit             - 提交报销单")
	log.Println("  GET    /api/v1/expense/:id                - 获取报销单详情")
	log.Println("  GET    /api/v1/expense/list               - 获取报销单列表")
	log.Println("  GET    /api/v1/department/list            - 获取部门列表")
	log.Println("  GET    /api/v1/department/budget/:id      - 获取部门预算")

	err := r.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
