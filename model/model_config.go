package model

import (
	"context"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
)

type ModelConfig struct {
	Model           string  `json:"model" gorm:"primaryKey"`
	Developer       string  `json:"developer"`
	DeveloperId     int32   `json:"developer_id"`
	ProviderId      int32   `json:"provider_id"`
	ModelName       string  `json:"model_name"`
	ModelRatio      float64 `json:"model_ratio"`
	CacheRatio      float64 `json:"cache_ratio"`
	CompletionRatio float64 `json:"completion_ratio"`
	Desc            string  `json:"desc"`
}

type ModelProvider struct {
	Id       int    `json:"id"`
	Provider string `json:"provider"`
	Color    string `json:"color"`
	Desc     string `json:"desc"`
}

type ModelDeveloper struct {
	Id        int    `json:"id"`
	Developer string `json:"developer"`
	Order     int    `json:"order"`
	Flag      int    `json:"flag"`
	Icon      string `json:"icon"`
	Desc      string `json:"desc"`
}

func InitModelConfig() {
	RefreshModelConfigCache(context.Background())
}

func RefreshModelConfigCache(ctx context.Context) {
	models, err := GetAllModelConfig(ctx)
	if err != nil {
		logger.Errorf(ctx, "failed to get all model config: %v", err)
		return
	}
	for _, model := range models {
		ratio.RefreshModelConfigCache(ctx, model.Model, model.ModelRatio, model.CacheRatio, model.CompletionRatio)
	}
}

func GetAllModelConfig(ctx context.Context) ([]*ModelConfig, error) {
	var modelConfigs []*ModelConfig
	err := DB.Find(&modelConfigs).Error
	return modelConfigs, err
}

func GetModelConfig(ctx context.Context, model string) (*ModelConfig, error) {
	modelConfig := ModelConfig{}
	err := DB.First(&modelConfig, "model = ?", model).Error
	return &modelConfig, err
}

func SaveModelConfig(ctx context.Context, modelConfig *ModelConfig) error {
	err := DB.Save(modelConfig).Error
	if err != nil {
		return err
	}
	ratio.RefreshModelConfigCache(ctx, modelConfig.Model, modelConfig.ModelRatio, modelConfig.CacheRatio, modelConfig.CompletionRatio)
	return nil
}

func DeleteModelConfig(ctx context.Context, model string) error {
	err := DB.Delete(&ModelConfig{}, "model = ?", model).Error
	if err != nil {
		return err
	}
	ratio.RefreshModelConfigCache(ctx, model, -1, -1, -1)
	return err
}

func GetAllModelProvider(ctx context.Context) ([]*ModelProvider, error) {
	var modelProviders []*ModelProvider
	err := DB.Find(&modelProviders).Error
	return modelProviders, err
}

func SaveModelProvider(ctx context.Context, modelProvider *ModelProvider) error {
	if modelProvider.Id == 0 {
		create := &ModelProvider{
			Provider: modelProvider.Provider,
			Color:    modelProvider.Color,
		}
		err := DB.Create(create).Error
		return err
	}
	err := DB.Save(modelProvider).Error
	return err
}

func GetAllModelDeveloper(ctx context.Context) ([]*ModelDeveloper, error) {
	var modelDevelopers []*ModelDeveloper
	err := DB.Find(&modelDevelopers).Error
	return modelDevelopers, err
}

func SaveModelDeveloper(ctx context.Context, modelDeveloper *ModelDeveloper) error {
	if modelDeveloper.Id == 0 {
		create := &ModelDeveloper{
			Developer: modelDeveloper.Developer,
			Order:     modelDeveloper.Order,
			Flag:      modelDeveloper.Flag,
			Icon:      modelDeveloper.Icon,
			Desc:      modelDeveloper.Desc,
		}
		err := DB.Create(create).Error
		return err
	}
	err := DB.Save(modelDeveloper).Error
	return err
}
