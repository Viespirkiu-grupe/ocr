package model

type Response struct {
	ID   int      `json:"id"`
	Text []string `json:"tekstas"`
	// milliseconds
	Duration int64 `json:"duration"`
}
