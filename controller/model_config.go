package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

func GetModelOptions(context *gin.Context) {
	ctx := context.Request.Context()
	modelConfigs, err := model.GetAllModelConfig(ctx)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	context.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    modelConfigs,
	})
}

func UpsertModelOption(context *gin.Context) {
	ctx := context.Request.Context()
	modelConfig := model.ModelConfig{}
	err := context.BindJSON(&modelConfig)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = model.SaveModelConfig(ctx, &modelConfig)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	context.JSON(200, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteModelOption(context *gin.Context) {
	ctx := context.Request.Context()
	m := context.Query("model")
	err := model.DeleteModelConfig(ctx, m)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	context.JSON(200, gin.H{
		"success": true,
		"message": "",
	})
}
