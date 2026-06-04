package scraper

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/datnguyen305/news_web/models"
	"github.com/datnguyen305/news_web/repository"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"
	"github.com/mmcdole/gofeed"
)

type ArticleSaver interface {
	SaveArticle(ctx context.Context, a models.Article) error
}

// ExtractThumbnail lấy link ảnh từ chuỗi Description của VnExpress
func ExtractThumbnail(desc string) string {
	// VnExpress thường để ảnh trong thẻ <img src="...">
	re := regexp.MustCompile(`src="([^"]+)"`)
	matches := re.FindStringSubmatch(desc)
	if len(matches) > 1 {
		return matches[1]
	}
	return "https://via.placeholder.com/600x400" // Ảnh mặc định nếu không tìm thấy
}

func ExtractDescription(desc string) string {
	// 1. Xóa tất cả các thẻ HTML (ví dụ: <a>, <img>, <br>)
	// Regex này tìm mọi thứ nằm giữa cặp dấu < >
	re := regexp.MustCompile(`<[^>]*>`)
	cleanText := re.ReplaceAllString(desc, "")

	// 2. Thay thế các thực thể HTML phổ biến (nếu có)
	// Ví dụ: &nbsp; thành khoảng trắng, &amp; thành &
	cleanText = strings.ReplaceAll(cleanText, "&nbsp;", " ")
	cleanText = strings.ReplaceAll(cleanText, "&amp;", "&")

	// 3. Loại bỏ khoảng trắng thừa ở hai đầu
	return strings.TrimSpace(cleanText)
}

func ScrapFullContent(ctx context.Context, url string) string {
	// 1. Tạo request với Timeout để tránh treo chương trình
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("Link bài viết không hợp lệ", "url", url, "err", err)
		return ""
	}

	res, err := client.Do(req)
	if err != nil {
		log.Error("Không thể truy cập link bài viết", "url", url, "err", err)
		return ""
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Error("Lỗi StatusCode", "status", res.StatusCode, "url", url)
		return ""
	}

	// 2. Load tài liệu HTML
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Error("Lỗi đọc HTML", "err", err)
		return ""
	}

	// 3. Chọn vùng chứa nội dung (Selector)
	// VnExpress thường để nội dung trong class .fck_detail hoặc bài podcast/video thì khác
	var contentBuilder strings.Builder

	// Thử selector phổ biến của VnExpress
	doc.Find("article.fck_detail p.Normal").Each(func(i int, s *goquery.Selection) {
		// Lấy text của từng đoạn văn và bọc vào thẻ <p>
		paragraph := s.Text()
		if strings.TrimSpace(paragraph) != "" {
			contentBuilder.WriteString("<p class='mb-4'>" + template.HTMLEscapeString(paragraph) + "</p>")
		}
	})

	return contentBuilder.String()
}

func Run(ctx context.Context, repo *repository.ArticleRepository, rssURL string, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}

	for {
		log.Info("Đang kiểm tra tin tức mới...")
		FetchAndSave(ctx, repo, rssURL)

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Info("Dừng scraper")
			return
		case <-timer.C:
		}
	}
}

func FetchAndSave(ctx context.Context, repo ArticleSaver, rssURL string) {
	fp := gofeed.NewParser()

	// Lấy URL từ file .env đã cấu hình
	if rssURL == "" {
		log.Error("Chưa cấu hình SCRAP_URL trong .env")
		return
	}

	feed, err := fp.ParseURL(rssURL)
	if err != nil {
		log.Error("Lỗi parse RSS", "url", rssURL, "err", err)
		return
	}

	for _, item := range feed.Items {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 1. Cào nội dung chi tiết (hàm ScrapFullContent của bạn)
		fullContent := ScrapFullContent(ctx, item.Link)

		// Nghỉ 1-2 giây để tránh bị VnExpress chặn IP
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		// 2. Tạo đối tượng Article
		article := models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Content:     template.HTML(fullContent),
			Description: ExtractDescription(item.Description),
			Thumbnail:   ExtractThumbnail(item.Description),
			Source:      "VnExpress",
			PublishedAt: publishedAtForItem(item, time.Now()),
		}

		// 3. Lưu vào DB thông qua Repository
		err := repo.SaveArticle(ctx, article)
		if err != nil {
			log.Error("Lỗi lưu bài viết", "title", article.Title, "err", err)
		} else {
			fmt.Printf("✅ Đã cập nhật: %s\n", article.Title)
		}
	}
}

func publishedAtForItem(item *gofeed.Item, fallback time.Time) time.Time {
	if item != nil && item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	return fallback
}
