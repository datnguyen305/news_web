package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"
	"webs/repository"

	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArticleHandler struct {
	DB        *pgxpool.Pool
	Templates map[string]*template.Template
	Repo      *repository.ArticleRepository // Thêm dòng này
}

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
	}
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
	articles, err := h.Repo.GetLatestArticles(10)
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
	id := r.URL.Query().Get("id")
	article, err := h.Repo.GetArticleByID(id)
	if err != nil {
		log.Error("Lỗi DB trang chi tiết", "id", id, "err", err)
		http.NotFound(w, r)
		return
	}

	// Sử dụng helper để lắp: layout.html + detail.html
	h.render(w, "detail", article)
}

func (h *ArticleHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	userMsg := r.FormValue("message")
	ai_url := os.Getenv("AI_SERVICE_URL")

	// 1. Gọi Python lấy ID
	reqPayload, _ := json.Marshal(map[string]string{"message": userMsg})
	resp, err := http.Post(ai_url, "application/json", bytes.NewBuffer(reqPayload))
	if err != nil { /* handle error */
		return
	}
	defer resp.Body.Close()

	var pyRes struct {
		ArticleIDs []int `json:"article_ids"`
	}
	json.NewDecoder(resp.Body).Decode(&pyRes)

	// 2. Query Postgres để lấy Title, Snippet và ImageURL từ những ID này
	articles, err := h.Repo.GetArticlesByIDs(pyRes.ArticleIDs) // Đổi từ h.repository thành h.Repo
	if err != nil {
		log.Error("Lỗi lấy bài báo từ repo", "err", err)
		http.Error(w, "Lỗi truy vấn dữ liệu", 500)
		return
	}

	// 3. Trả về JSON đầy đủ cho Frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"articles": articles,
	})
}
