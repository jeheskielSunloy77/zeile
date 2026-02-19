package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
)

func PrintJSON(v any) {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}
	fmt.Println("JSON:", string(json))
}

func GetModelName[T domain.BaseModel]() string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func GetModelNameLower[T domain.BaseModel]() string {
	return strings.ToLower(GetModelName[T]())
}

func GetModelSemanticName[T domain.BaseModel]() string {
	var name strings.Builder

	for i, r := range GetModelName[T]() {
		if i > 0 && r >= 'A' && r <= 'Z' {
			name.WriteRune(' ')
		}
		name.WriteRune(r)
	}
	return name.String()

}
func GetModelCacheKey[T domain.BaseModel](id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return "resource:" + GetModelNameLower[T]() + ":id:" + id.String()
}
