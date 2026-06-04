package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/datnguyen305/news_web/models"
	"github.com/datnguyen305/news_web/repository"

	"github.com/charmbracelet/log"
)

type ArticleStore interface {
	GetLatestArticles(ctx context.Context, limit int) ([]models.Article, error)
	GetArticleByID(ctx context.Context, id string) (models.Article, error)
	GetArticlesByIDs(ctx context.Context, ids []int) ([]models.RelatedArticle, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ArticleHandler struct {
	Templates    map[string]*template.Template
	Repo         ArticleStore
	AIServiceURL string
	HTTPClient   HTTPDoer
}

var _ ArticleStore = (*repository.ArticleRepository)(nil)

func TimeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration.Minutes() < 1:
		return "Vừa xong"
	case duration.Minutes() < 60:
		return fmt.Sprintf("%.0f phút trước", duration.Minutes())
	case duration.Hours() < 24:
		return fmt.Sprintf("%.0f giờ trước", duration.Hours())
	default:
		return t.Format("02/01/2006")
	}
}

func GetFuncMap() template.FuncMap {
	return template.FuncMap{
		"timeAgo": TimeAgo,
		"slice":   Slice,
	}
}

func Slice(value interface{}, start int) (interface{}, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("slice expects slice or array, got %T", value)
	}
	if start < 0 || start > v.Len() {
		return nil, fmt.Errorf("slice start %d out of range", start)
	}
	return v.Slice(start, v.Len()).Interface(), nil
}

// Hàm helper để render (Để ở ngoài hoặc trong struct đều được)
func (h *ArticleHandler) render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := h.Templates[name]
	if !ok {
		log.Error("Template không tồn tại", "name", name)
		http.Error(w, "Lỗi hệ thống", 500)
		return
	}

	err := tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Error("Lỗi Render", "name", name, "err", err)
		http.Error(w, err.Error(), 500)
	}
}

// Home xử lý trang chủ
func (h *ArticleHandler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	articles, err := h.Repo.GetLatestArticles(r.Context(), 10)
	if err != nil {
		log.Error("Lỗi truy vấn trang chủ", "err", err)
		http.Error(w, "Lỗi server", 500)
		return
	}

	// Sử dụng helper để lắp: layout.html + index.html
	h.render(w, "index", articles)
}

// Detail xử lý trang chi tiết
func (h *ArticleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Thiếu id bài viết", http.StatusBadRequest)
		return
	}

	article, err := h.Repo.GetArticleByID(r.Context(), id)
	if err != nil {
		log.Error("Lỗi DB trang chi tiết", "id", id, "err", err)
		http.NotFound(w, r)
		return
	}

	// Sử dụng helper để lắp: layout.html + detail.html
	h.render(w, "detail", article)
}

func (h *ArticleHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Chỉ hỗ trợ POST")
		return
	}

	userMsg := r.FormValue("message")
	if userMsg == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_message", "Vui lòng nhập câu hỏi")
		return
	}
	if h.AIServiceURL == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "ai_service_unconfigured", "AI service chưa được cấu hình")
		return
	}

	// 1. Gọi Python lấy ID
	reqPayload, err := json.Marshal(map[string]string{"message": userMsg})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "encode_failed", "Không thể xử lý câu hỏi")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.AIServiceURL, bytes.NewBuffer(reqPayload))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "bad_ai_url", "AI service URL không hợp lệ")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := h.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Lỗi gọi AI service", "err", err)
		writeJSONError(w, http.StatusBadGateway, "ai_request_failed", "Không thể kết nối AI service")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		log.Error("AI service trả về lỗi", "status", resp.StatusCode)
		writeJSONError(w, http.StatusBadGateway, "ai_bad_status", "AI service trả về lỗi")
		return
	}

	var pyRes struct {
		ArticleIDs []int `json:"article_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pyRes); err != nil {
		log.Error("Lỗi đọc JSON từ AI service", "err", err)
		writeJSONError(w, http.StatusBadGateway, "ai_bad_response", "AI service trả về dữ liệu không hợp lệ")
		return
	}

	// 2. Query Postgres để lấy Title, Snippet và ImageURL từ những ID này
	articles, err := h.Repo.GetArticlesByIDs(r.Context(), pyRes.ArticleIDs)
	if err != nil {
		log.Error("Lỗi lấy bài báo từ repo", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "article_lookup_failed", "Lỗi truy vấn dữ liệu")
		return
	}

	// 3. Trả về JSON đầy đủ cho Frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"articles": articles,
	})
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
