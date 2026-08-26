package model

func GetModelEnableGroups(modelName string) []string {
	if modelName == "" {
		return make([]string, 0)
	}
	return []string{"all"}
}
