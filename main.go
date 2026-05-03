package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/datnguyen305/news_web/database"
	"github.com/datnguyen305/news_web/handlers" // Import package handlers mới
	"github.com/datnguyen305/news_web/repository"
	"github.com/datnguyen305/news_web/scraper"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	// ... các import khác
)

func initTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	// ĐỊNH NGHĨA CÁC FUNC MAP (nếu bạn có dùng hàm như timeAgo)
	funcMap := handlers.GetFuncMap()

	// 1. Template cho trang chủ: layout.html + index.html
	templates["index"] = template.Must(template.New("layout").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/index.html",
	))

	// 2. Template cho trang chi tiết: layout.html + detail.html
	templates["detail"] = template.Must(template.New("layout").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/detail.html",
	))

	return templates
}

func main() {
	// 1. Khởi tạo kết nối Database (ví dụ pgxpool)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Lỗi: Không tìm thấy file .env")
	}
	dbPool := database.InitDB()
	defer dbPool.Close()

	myTemplates := initTemplates()

	// 2. KHỞI TẠO REPOSITORY TRƯỚC
	// Đảm bảo bạn đã truyền dbPool vào struct này
	articleRepo := &repository.ArticleRepository{
		Pool: dbPool,
	}

	go func() {
		for {
			log.Info("🔄 Đang kiểm tra tin tức mới...")
			scraper.FetchAndSave(articleRepo)

			log.Info("💤 Ngủ 1 giờ trước chu kỳ tiếp theo")
			time.Sleep(1 * time.Hour)
		}
	}()

	// 3. TRUYỀN REPOSITORY VÀO HANDLER
	articleHdl := &handlers.ArticleHandler{
		DB:        dbPool,
		Templates: myTemplates, // Map templates của bạn
		Repo:      articleRepo, // CỰC KỲ QUAN TRỌNG: Không được để trống dòng này
	}

	// 4. Đăng ký các route
	http.HandleFunc("/", articleHdl.Home)
	http.HandleFunc("/detail", articleHdl.Detail)
	http.HandleFunc("/chat", articleHdl.HandleChat)

	// Chạy server...
	log.Info("Server đang chạy tại http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
