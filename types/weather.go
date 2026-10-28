package types

type WeatherResult struct {
	City      string `json:"city"`
	Condition string `json:"condition"`
	Temp      int    `json:"temp"`
}

type WeatherParams struct {
	City string `json:"city" jsonschema:"City to get weather for"`
}
