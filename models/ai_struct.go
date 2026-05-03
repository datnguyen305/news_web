package models

type AIChatRequest struct {
	Message string `json:"message"`
}

type RelatedArticle struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Snippet  string `json:"snippet"`
	ImageURL string `json:"image_url"` // Thêm trường này
}

type AIChatResponse struct {
	Articles []RelatedArticle `json:"articles"`
}
