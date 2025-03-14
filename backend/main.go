package main

import (
	"context"
	"fruit-manager-system/db"
	"fruit-manager-system/middlewares"
	"fruit-manager-system/routes"
	"github.com/labstack/echo/v4"
	"log"
	"time"
)

func main() {
	db.InitMongo("mongodb://localhost:27017")
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.Client.Disconnect(ctx); err != nil {
			log.Printf("關閉連線時發生錯誤: %v", err)
		}
	}()

	database := db.Client.Database("fruit-manager-system")

	e := echo.New()

	e.Use(middlewares.DatabaseMiddleware(database))

	routes.RegisterRoutes(e)

	e.Logger.Fatal(e.Start(":8880"))
}
