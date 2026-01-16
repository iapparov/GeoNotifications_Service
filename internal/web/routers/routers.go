package routers

import (
	_ "geoNotifications/docs"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/web/handlers"
	"geoNotifications/internal/web/middlewares"

	"github.com/gin-gonic/gin"
	httpSwagger "github.com/swaggo/http-swagger"
)

func RegisterRoutes(engine *gin.Engine, hLocation *handlers.Location, hIncidents *handlers.Incidents, hSystem *handlers.System, l *logger.Service, key string) {

	engine.Use(gin.Recovery())
	engine.Use(middlewares.LoggerMiddleware(l))

	// Register routes
	api := engine.Group("/api/v1")
	api.GET("/swagger/*filepath", gin.WrapH(httpSwagger.WrapHandler))

	incidents := api.Group("/incidents")
	location := api.Group("/location")
	system := api.Group("/system")

	incidents.GET("/stats", hIncidents.GetStats)
	incidents.Use(middlewares.AuthMiddleware(key))
	incidents.POST("", hIncidents.Create)
	incidents.GET("", hIncidents.GetAllIncidents)
	incidents.GET("/:id", hIncidents.GetIncidentByID)
	incidents.PUT("/:id", hIncidents.PutIncident)
	incidents.DELETE("/:id", hIncidents.DeleteIncident)

	location.POST("/check", hLocation.Check)

	system.GET("/health", hSystem.Health)
}
