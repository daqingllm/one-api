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
	DescEn          string  `json:"desc_en"`
	Order           int     `json:"order"`
	Flag            int     `json:"flag"`
	ContextLength   string  `json:"context_length"`
	ParametersCount string  `json:"parameters_count"`
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
	Icon      string `json:"icon"`
	Desc      string `json:"desc"`
}

type ModelTag struct {
	Id    int    `json:"id" gorm:"primaryKey"`
	Model string `json:"model" gorm:"index"`
	TagID int    `json:"tag_id"`
}

type Tag struct {
	Id     int    `json:"id" gorm:"primaryKey"`
	Name   string `json:"name"`
	NameEn string `json:"name_en"`
	Type   int    `json:"type"`
}
type ModelParameter struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	Model        string `json:"model" gorm:"index"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value"`
	Options      string `json:"options"`
	Desc         string `json:"desc"`
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
	if err != nil {
		return nil, err
	}
	return &modelConfig, err
}

func GetModelTags(ctx context.Context, model string) ([]Tag, error) {
	var modelTags []*ModelTag
	err := DB.Find(&modelTags, "model = ?", model).Error
	if err != nil {
		return nil, err
	}
	var tags []Tag
	for _, modelTag := range modelTags {
		tag := Tag{}
		err := DB.First(&tag, "id = ?", modelTag.TagID).Error
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, err
}

func GetModelTagsRelative(ctx context.Context, model string) ([]*ModelTag, error) {
	var modelTags []*ModelTag
	err := DB.Find(&modelTags, "model = ?", model).Error
	if err != nil {
		return nil, err
	}
	return modelTags, err
}

func SaveModelTag(ctx context.Context, modelTag *ModelTag) error {
	if modelTag.Id == 0 {
		create := &ModelTag{
			Model: modelTag.Model,
			TagID: modelTag.TagID,
		}
		err := DB.Create(create).Error
		return err
	}
	err := DB.Save(modelTag).Error
	return err
}

func DeleteModelTag(ctx context.Context, Id int) error {
	err := DB.Delete(&ModelTag{}, "id = ?", Id).Error
	if err != nil {
		return err
	}
	return err
}

func GetModelParameters(ctx context.Context, model string) ([]*ModelParameter, error) {
	var modelParameters []*ModelParameter
	err := DB.Find(&modelParameters, "model = ?", model).Error
	if err != nil {
		return nil, err
	}
	return modelParameters, err
}

func SaveModelParameter(ctx context.Context, modelParameter *ModelParameter) error {
	if modelParameter.Id == 0 {
		create := &ModelParameter{
			Model:        modelParameter.Model,
			Name:         modelParameter.Name,
			Type:         modelParameter.Type,
			Required:     modelParameter.Required,
			DefaultValue: modelParameter.DefaultValue,
			Options:      modelParameter.Options,
			Desc:         modelParameter.Desc,
		}
		err := DB.Create(create).Error
		return err
	}
	err := DB.Save(modelParameter).Error
	return err
}

func DeleteModelParameter(ctx context.Context, Id int) error {
	err := DB.Delete(&ModelParameter{}, "id = ?", Id).Error
	if err != nil {
		return err
	}
	return err
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
			Icon:      modelDeveloper.Icon,
			Desc:      modelDeveloper.Desc,
		}
		err := DB.Create(create).Error
		return err
	}
	err := DB.Save(modelDeveloper).Error
	return err
}

func GetFixedTags(ctx context.Context) ([]*Tag, error) {
	var tags []*Tag
	err := DB.Find(&tags, "type = ?", 1).Error
	return tags, err
}

func GetAllTags(ctx context.Context) ([]*Tag, error) {
	var tags []*Tag
	err := DB.Find(&tags).Error
	return tags, err
}
