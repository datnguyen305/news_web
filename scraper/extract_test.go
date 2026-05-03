package scraper

import (
	"testing"
)

func TestExtractDescription(t *testing.T) {
	input := `<a href="..."><img src="..."></a>Nội dung tóm tắt bài báo &nbsp; và các thẻ khác.`
	expected := "Nội dung tóm tắt bài báo   và các thẻ khác."

	result := ExtractDescription(input)

	if result != expected {
		t.Errorf("ExtractDescription() lỗi!\nKết quả: %v\nMong đợi: %v", result, expected)
	}
}

func TestExtractThumbnail(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		expected string
	}{
		{
			name:     "Có ảnh hợp lệ",
			desc:     `<img src="https://vcdn-vnexpress.vnecdn.net/image.jpg">`,
			expected: "https://vcdn-vnexpress.vnecdn.net/image.jpg",
		},
		{
			name:     "Không có ảnh",
			desc:     `Chỉ có nội dung chữ`,
			expected: "https://via.placeholder.com/600x400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractThumbnail(tt.desc)
			if result != tt.expected {
				t.Errorf("ExtractThumbnail() = %v, want %v", result, tt.expected)
			}
		})
	}
}
