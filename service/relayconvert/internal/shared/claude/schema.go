package claude

// FunctionParametersToInputSchema normalizes an OpenAI function parameter
// declaration into the object schema required by Claude. Missing parameters
// are represented as an empty object schema so parameterless tools are kept.
func FunctionParametersToInputSchema(parameters any) map[string]any {
	params, _ := parameters.(map[string]any)
	schema := make(map[string]any, len(params)+2)
	for key, value := range params {
		schema[key] = value
	}
	if schema["type"] == nil {
		schema["type"] = "object"
	}
	if schema["properties"] == nil {
		schema["properties"] = map[string]any{}
	}
	return schema
}
