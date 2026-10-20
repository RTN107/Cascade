package main

import (
	"context"
	"log"
	"net/http"

	"cascade/internal/audit"
	"cascade/internal/config"
	"cascade/internal/db"
	"cascade/internal/employees"
	"cascade/internal/httpapi"
	"cascade/internal/leave"
	"cascade/internal/payroll"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	store := employees.NewStore(pool)
	auditStore := audit.NewStore(pool)
	leaveStore := leave.NewStore(pool)
	payrollStore := payroll.NewStore(pool)
	handler := httpapi.NewRouter(store, auditStore, leaveStore, payrollStore, cfg.JWTSecret, cfg.WebOrigin)

	log.Printf("api listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}
