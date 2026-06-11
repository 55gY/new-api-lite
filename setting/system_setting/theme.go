package system_setting

type ThemeSettings struct {
	Frontend string `json:"frontend"`
}

var themeSettings = ThemeSettings{
	Frontend: "classic",
}

func GetThemeSettings() *ThemeSettings {
	return &themeSettings
}
