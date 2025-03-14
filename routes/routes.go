package routes

import (
	"fruit-manager-system/controllers"
	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	e.POST("/fruits/order", controllers.HandleOrderFruits)
	e.POST("/fruits/restock", controllers.HandleRestock)
	e.GET("/fruits", controllers.HandleSearchStocks)
	e.PUT("/fruits", controllers.HandleUpdateStock)
}
