package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

type EnhancedModelConfigOperate struct {
	model.ModelConfig
	Tags             []*model.ModelTag       `json:"tags"`
	Parameters       []*model.ModelParameter `json:"parameters"`
	DeleteTags       []int                   `json:"delete_tags"`
	DeleteParameters []int                   `json:"delete_parameters"`
}

// 统一处理错误
func handleError(context *gin.Context, statusCode int, err error) {
	context.JSON(statusCode, gin.H{
		"success": false,
		"message": err.Error(),
	})
}

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
	EnhancedModels := []EnhancedModelConfigOperate{}
	// 循环遍历models，获取每个model的tags和parameters
	for _, m := range modelConfigs {
		tags, err := model.GetModelTagsRelative(ctx, m.Model)
		if err != nil {
			context.JSON(200, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		parameters, err := model.GetModelParameters(ctx, m.Model)
		if err != nil {
			context.JSON(200, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		EnhancedModels = append(EnhancedModels, EnhancedModelConfigOperate{
			ModelConfig: *m,
			Tags:        tags,
			Parameters:  parameters,
		})
	}
	context.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    EnhancedModels,
	})
}

// 处理标签的插入和删除
func UpsertTags(ctx *gin.Context, tags []*model.ModelTag, deleteTags []int) error {
	for _, tag := range tags {
		if err := model.SaveModelTag(ctx, tag); err != nil {
			return err
		}
	}
	for _, tag := range deleteTags {
		if err := model.DeleteModelTag(ctx, tag); err != nil {
			return err
		}
	}
	return nil
}

// 处理参数的插入和删除
func UpsertParameters(ctx *gin.Context, parameters []*model.ModelParameter, deleteParameters []int) error {
	for _, parameter := range parameters {
		if err := model.SaveModelParameter(ctx, parameter); err != nil {
			return err
		}
	}
	for _, parameter := range deleteParameters {
		if err := model.DeleteModelParameter(ctx, parameter); err != nil {
			return err
		}
	}
	return nil
}

// 处理模型配置的插入和更新
func UpsertModelOption(context *gin.Context) {
	ctx := context.Request.Context()
	modelConfig := EnhancedModelConfigOperate{}
	err := context.BindJSON(&modelConfig)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 保存模型配置
	if err := model.SaveModelConfig(ctx, &modelConfig.ModelConfig); err != nil {
		handleError(context, 200, err) // 使用 500 Internal Server Error
		return
	}

	// 处理标签
	if err := UpsertTags(context, modelConfig.Tags, modelConfig.DeleteTags); err != nil {
		handleError(context, 200, err)
		return
	}

	// 处理参数
	if err := UpsertParameters(context, modelConfig.Parameters, modelConfig.DeleteParameters); err != nil {
		handleError(context, 200, err)
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

func GetChannelProviders(context *gin.Context) {
	ctx := context.Request.Context()
	modelProviders, err := model.GetAllModelProvider(ctx)
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
		"data":    modelProviders,
	})
}

func AddChannelProvider(context *gin.Context) {
	ctx := context.Request.Context()
	modelProvider := model.ModelProvider{}
	err := context.BindJSON(&modelProvider)
	if err != nil {
		context.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = model.SaveModelProvider(ctx, &modelProvider)
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

func GetAllTags(context *gin.Context) {
	ctx := context.Request.Context()
	tags, err := model.GetAllTags(ctx)
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
		"data":    tags,
	})
}
