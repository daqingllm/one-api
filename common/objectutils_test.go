package common

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/songquanpeng/one-api/relay/model"
)

func TestRecursivelyCheckAndAssign(t *testing.T) {
	jsonData := `{
      "type": "function",
      "function": {
        "name": "f825a0517dbaa41119b5b309f49bd80ac",
        "description": "Get the current time in the configured local timezone",
        "parameters": {
						"type": "object",
						"properties":{
						}
          }
      }
    }`
	var tool *model.Tool

	err := json.Unmarshal([]byte(jsonData), &tool)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	RecursivelyCheckAndAssign(&tool.Function)

	fmt.Println("Before:", tool)

	// SetNilForEmptyObjects1(tool)
	// 序列化为 JSON 输出
	result, _ := json.MarshalIndent(tool, "", "  ")
	fmt.Println("After:", string(result))
}
