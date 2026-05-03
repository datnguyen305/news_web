package models

import (
	"html/template"
	"time"
)

type Article struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Content     template.HTML // Dùng để hiển thị HTML từ RSS không bị lỗi  // Nội dung đầy đủ cho trang chi tiết
	Thumbnail   string        `json:"thumbnail"` // Link ảnh đại diện
	Source      string        `json:"source"`    // Nguồn (VnExpress, Tuổi Trẻ...)
	Link        string        `json:"link"`      // Link gốc của bài báo
	PublishedAt time.Time
	Category    string `json:"category"`
}