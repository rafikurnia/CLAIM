package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/rafikurnia/measurement-measurer/utils"
)

func SetupRouter() (*gin.Engine, error) {
	firestoreCollectionName = os.Getenv("FIRESTORE_COLLECTION_NAME")

	router := gin.Default()
	router.HandleMethodNotAllowed = true

	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("router.SetTrustedProxies -> %w", err)
	}

	v1 := router.Group("/api/v1")
	{
		v1.POST("/measurements", runMeasurement)
	}

	router.NoRoute(func(ctx *gin.Context) {
		utils.Throws(ctx, http.StatusNotFound, http.StatusText(http.StatusNotFound))
		return
	})
	router.NoMethod(func(ctx *gin.Context) {
		utils.Throws(ctx, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		return
	})

	return router, nil
}
