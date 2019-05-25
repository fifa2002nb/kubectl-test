package controllers

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"kubectl-test/app/models"
	"net/http"
)

func TestEndpoint(c *gin.Context) {
	var kube_req *models.KubectlPluginRequest = &models.KubectlPluginRequest{}
	var err error
	err = c.BindJSON(kube_req)
	var ret interface{}
	if nil != err {
		log.Errorf("%v", err)
	} else {
		ret = kube_req
	}
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, ret)
	return
}

func HealthEndpoint(c *gin.Context) {
	var kube_req *models.KubectlPluginRequest = &models.KubectlPluginRequest{}
	var err error
	err = c.BindJSON(kube_req)
	var ret interface{}
	if nil != err {
		log.Errorf("%v", err)
	} else {
		ret = kube_req
	}
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, ret)
	return
}
