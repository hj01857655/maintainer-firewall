package handlers

import "github.com/gin-gonic/gin"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Health(c *gin.Context) {
	c.JSON(200, HealthResponse{
		Status:  "ok",
		Service: "maintainer-firewall-api",
	})
}
