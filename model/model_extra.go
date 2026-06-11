package model

func GetModelEnableGroups(modelName string) []string {
	if modelName == "" {
		return make([]string, 0)
	}
	return []string{"all"}
}

// GetModelQuotaTypes 保留兼容字段，计费能力移除后不再返回计费类型。
func GetModelQuotaTypes(modelName string) []int {
	return []int{}
}
