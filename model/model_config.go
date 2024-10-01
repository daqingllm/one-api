package model

import (
	"context"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
)

type ModelConfig struct {
	Model           string  `json:"model" gorm:"primaryKey"`
	Developer       string  `json:"developer"`
	Provider        string  `json:"provider"`
	ModelName       string  `json:"model_type"`
	ModelRatio      float64 `json:"model_ratio"`
	CompletionRatio float64 `json:"completion_ratio"`
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
		ratio.RefreshModelConfigCache(ctx, model.Model, model.ModelRatio, model.CompletionRatio)
	}
}

func GetAllModelConfig(ctx context.Context) ([]*ModelConfig, error) {
	var modelConfigs []*ModelConfig
	var err error
	err = DB.Find(&modelConfigs).Error
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
	ratio.RefreshModelConfigCache(ctx, modelConfig.Model, modelConfig.ModelRatio, modelConfig.CompletionRatio)
	return nil
}

func DeleteModelConfig(ctx context.Context, model string) error {
	err := DB.Delete(&ModelConfig{}, "model = ?", model).Error
	if err != nil {
		return err
	}
	ratio.RefreshModelConfigCache(ctx, model, 0, 0)
	return err
}
