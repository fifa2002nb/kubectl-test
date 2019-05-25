package routes

import (
	"github.com/gin-gonic/gin"
	"kubectl-test/app/controllers"
	"net/http"
)

func Router() *http.ServeMux {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	v1 := router.Group("/v1")
	{
		v1.POST("/api/test", controllers.TestEndpoint)
		v1.POST("/api/health", controllers.HealthEndpoint)
	}
	var mux = http.NewServeMux()
	mux.Handle("/", router)
	return mux
}
