package repository

import (
	"context"
	"database/sql"
	"html/template"
	"webs/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq" // Đừng quên import pq để dùng cho hàm ANY($1)
)

// BẮT BUỘC: Định nghĩa struct này để Handler có thể nhận diện
type ArticleRepository struct {
	Pool *pgxpool.Pool
}

// Hàm khởi tạo Repository (Constructor)
func NewArticleRepository(pool *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{Pool: pool}
}

// Chuyển SaveArticle thành method của ArticleRepository
func (r *ArticleRepository) SaveArticle(a models.Article) error {
	query := `INSERT INTO articles (title, link, description, content, thumbnail, source, published_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)
              ON CONFLICT (link) DO UPDATE 
              SET content = EXCLUDED.content 
              WHERE articles.content IS NULL OR articles.content = ''`

	_, err := r.Pool.Exec(context.Background(), query,
		a.Title, a.Link, a.Description, string(a.Content), a.Thumbnail, a.Source, a.PublishedAt,
	)
	return err
}

// Chuyển GetLatestArticles thành method
func (r *ArticleRepository) GetLatestArticles(limit int) ([]models.Article, error) {
	query := `SELECT id, title, link, description, thumbnail, source, published_at 
              FROM articles ORDER BY published_at DESC LIMIT $1`

	rows, err := r.Pool.Query(context.Background(), query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		err := rows.Scan(&a.ID, &a.Title, &a.Link, &a.Description, &a.Thumbnail, &a.Source, &a.PublishedAt)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, nil
}

// Chuyển GetArticleByID thành method
func (r *ArticleRepository) GetArticleByID(id string) (models.Article, error) {
	var a models.Article
	var contentNull sql.NullString

	query := `SELECT id, title, description, content, thumbnail, source, link, published_at 
              FROM articles WHERE id = $1`

	err := r.Pool.QueryRow(context.Background(), query, id).Scan(
		&a.ID, &a.Title, &a.Description, &contentNull, &a.Thumbnail, &a.Source, &a.Link, &a.PublishedAt,
	)

	if err != nil {
		return a, err
	}

	if contentNull.Valid {
		a.Content = template.HTML(contentNull.String)
	} else {
		a.Content = template.HTML("Nội dung đang được cập nhật...")
	}

	return a, nil
}

// ĐÂY LÀ HÀM BẠN CẦN CHO CHATBOT
func (r *ArticleRepository) GetArticlesByIDs(ids []int) ([]models.RelatedArticle, error) {
	if len(ids) == 0 {
		return []models.RelatedArticle{}, nil
	}

	query := `SELECT id, title, SUBSTRING(description, 1, 100), thumbnail 
              FROM articles WHERE id = ANY($1)`

	rows, err := r.Pool.Query(context.Background(), query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.RelatedArticle
	for rows.Next() {
		var a models.RelatedArticle
		// Scan vào struct RelatedArticle (nhớ kiểm tra struct này trong models)
		err := rows.Scan(&a.ID, &a.Title, &a.Snippet, &a.ImageURL)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}
