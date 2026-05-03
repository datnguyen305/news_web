package handlers

import (
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "Vừa xong",
			input:    now.Add(-30 * time.Second),
			expected: "Vừa xong",
		},
		{
			name:     "Phút trước",
			input:    now.Add(-5 * time.Minute),
			expected: "5 phút trước",
		},
		{
			name:     "Giờ trước",
			input:    now.Add(-3 * time.Hour),
			expected: "3 giờ trước",
		},
		{
			name:     "Định dạng ngày",
			input:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "01/01/2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeAgo(tt.input)
			if result != tt.expected {
				t.Errorf("TimeAgo() = %v, want %v", result, tt.expected)
			}
		})
	}
}
