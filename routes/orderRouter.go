package routes

import (
	"go-restaurant-managament/controllers"

	"github.com/gin-gonic/gin"
)

func OrderRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/orders", controllers.GetOrders())
	incomingRoutes.GET("/orders/:order_id", controllers.GetOrder())
	incomingRoutes.POST("/orders", controllers.CreatedOrder())
	incomingRoutes.PATCH("/orders/:order_id", controllers.UpdateOrder())
	incomingRoutes.GET("/orderswithitems", controllers.GetAllOrdersWithItem())
}
