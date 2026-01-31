package main

import (
	"log"
	"os"

	"github.com/donation-station/donation-station/internal/api"
	"github.com/donation-station/donation-station/internal/config"
	"github.com/donation-station/donation-station/internal/database"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	// 创建并启动服务器
	server := api.NewServer(cfg, db)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting donation station server on port %s...", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
