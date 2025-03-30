package common

import (
	"reflect"
)

func RecursivelyCheckAndAssign(ptr any) {
	value := reflect.ValueOf(ptr)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if IsEmptyObject(value) {
			value.Set(reflect.Zero(value.Type()))
			return
		}
		for i := 0; i < value.Len(); i++ {
			element := value.Index(i)
			RecursivelyCheckAndAssign(element.Addr().Interface())
		}
		// 新增：处理子元素后重新检查空对象
		if IsEmptyObject(value) {
			value.Set(reflect.Zero(value.Type()))
		}
	case reflect.Map:
		keysToDelete := make([]reflect.Value, 0)
		for _, key := range value.MapKeys() {
			element := value.MapIndex(key)
			// 创建指向元素的指针
			elemPtr := reflect.New(element.Type())
			elemPtr.Elem().Set(element)
			RecursivelyCheckAndAssign(elemPtr.Interface())
			newValue := elemPtr.Elem()
			value.SetMapIndex(key, newValue)

			// 标记需要删除的空对象键
			if IsEmptyObject(newValue.Interface()) {
				keysToDelete = append(keysToDelete, key)
			}
		}
		// 删除所有空对象键
		for _, key := range keysToDelete {
			value.SetMapIndex(key, reflect.Value{})
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			fieldValue := value.Field(i)

			// 显式处理 any 类型的字段（如 Function.Parameters）
			if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
				// 解包 interface{} 为实际类型
				elemValue := reflect.ValueOf(fieldValue.Interface())
				elemPtr := reflect.New(elemValue.Type())
				elemPtr.Elem().Set(elemValue)
				RecursivelyCheckAndAssign(elemPtr.Interface())
				fieldValue.Set(elemPtr.Elem())
			}

			if fieldValue.CanSet() && IsEmptyObject(fieldValue.Interface()) {
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}

			// 递归处理字段
			if fieldValue.CanAddr() {
				RecursivelyCheckAndAssign(fieldValue.Addr().Interface())
			}
		}
	}

}

func IsEmptyObject(obj any) bool {
	value := reflect.ValueOf(obj)
	return (value.Kind() == reflect.Map || value.Kind() == reflect.Array || value.Kind() == reflect.Slice) && value.Len() == 0
}
