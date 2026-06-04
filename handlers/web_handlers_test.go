package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datnguyen305/news_web/models"
)

type fakeStore struct {
	latest       []models.Article
	article      models.Article
	related      []models.RelatedArticle
	latestErr    error
	articleErr   error
	relatedErr   error
	requestedIDs []int
}

func (s *fakeStore) GetLatestArticles(ctx context.Context, limit int) ([]models.Article, error) {
	return s.latest, s.latestErr
}

func (s *fakeStore) GetArticleByID(ctx context.Context, id string) (models.Article, error) {
	return s.article, s.articleErr
}

func (s *fakeStore) GetArticlesByIDs(ctx context.Context, ids []int) ([]models.RelatedArticle, error) {
	s.requestedIDs = ids
	return s.related, s.relatedErr
}

type fakeHTTPClient struct {
	resp *http.Response
	err  error
}

func (c fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.resp, c.err
}

func TestSlice(t *testing.T) {
	result, err := Slice([]int{1, 2, 3}, 1)
	if err != nil {
		t.Fatalf("Slice() error = %v", err)
	}
	values := result.([]int)
	if len(values) != 2 || values[0] != 2 || values[1] != 3 {
		t.Fatalf("Slice() = %v, want [2 3]", values)
	}
}

func TestHandleChatRequiresPost(t *testing.T) {
	h := &ArticleHandler{Repo: &fakeStore{}}
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	h.HandleChat(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleChatRequiresMessage(t *testing.T) {
	h := &ArticleHandler{Repo: &fakeStore{}, AIServiceURL: "http://ai.test/chat"}
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.HandleChat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleChatRequiresAIServiceURL(t *testing.T) {
	h := &ArticleHandler{Repo: &fakeStore{}}
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("message=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.HandleChat(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleChatHandlesAIRequestFailure(t *testing.T) {
	h := &ArticleHandler{
		Repo:         &fakeStore{},
		AIServiceURL: "http://ai.test/chat",
		HTTPClient:   fakeHTTPClient{err: errors.New("network down")},
	}
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("message=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.HandleChat(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestHandleChatSuccess(t *testing.T) {
	store := &fakeStore{
		related: []models.RelatedArticle{{ID: 1, Title: "Title", Snippet: "Snippet", ImageURL: "image.jpg"}},
	}
	h := &ArticleHandler{
		Repo:         store,
		AIServiceURL: "http://ai.test/chat",
		HTTPClient: fakeHTTPClient{resp: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"article_ids":[1,2]}`)),
		}},
	}
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("message=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.HandleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(store.requestedIDs) != 2 || store.requestedIDs[0] != 1 || store.requestedIDs[1] != 2 {
		t.Fatalf("requestedIDs = %v, want [1 2]", store.requestedIDs)
	}
}

func TestHomeRejectsUnknownPath(t *testing.T) {
	h := &ArticleHandler{Repo: &fakeStore{}}
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDetailRequiresID(t *testing.T) {
	h := &ArticleHandler{Repo: &fakeStore{}}
	req := httptest.NewRequest(http.MethodGet, "/detail", nil)
	rec := httptest.NewRecorder()

	h.Detail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTimeAgoFutureIsJustNow(t *testing.T) {
	result := TimeAgo(time.Now().Add(1 * time.Minute))
	if result != "Vừa xong" {
		t.Fatalf("TimeAgo(future) = %q, want Vừa xong", result)
	}
}
