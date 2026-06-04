package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datnguyen305/news_web/database"
	"github.com/datnguyen305/news_web/handlers"
	"github.com/datnguyen305/news_web/repository"
	"github.com/datnguyen305/news_web/scraper"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
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
	if err := run(); err != nil {
		log.Fatal("Ứng dụng dừng do lỗi", "err", err)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbCtx, cancelDB := context.WithTimeout(rootCtx, 10*time.Second)
	defer cancelDB()

	dbPool, err := database.InitDB(dbCtx, cfg.DBURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	myTemplates := initTemplates()
	articleRepo := repository.NewArticleRepository(dbPool)

	if cfg.ScrapURL != "" {
		go scraper.Run(rootCtx, articleRepo, cfg.ScrapURL, cfg.ScraperInterval)
	} else {
		log.Warn("SCRAP_URL chưa được cấu hình, bỏ qua scraper nền")
	}

	articleHdl := &handlers.ArticleHandler{
		Templates:    myTemplates,
		Repo:         articleRepo,
		AIServiceURL: cfg.AIServiceURL,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", articleHdl.Home)
	mux.HandleFunc("/detail", articleHdl.Detail)
	mux.HandleFunc("/chat", articleHdl.HandleChat)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("Server đang chạy", "url", "http://localhost:"+cfg.Port)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}
