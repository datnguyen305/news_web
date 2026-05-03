import psycopg2
import os
from langchain_community.vectorstores import Chroma
from langchain_core.documents import Document
from sentence_transformers import SentenceTransformer
from langchain_text_splitters import RecursiveCharacterTextSplitter

# 1. Cấu hình
CONNECTION_STRING = "postgresql://postgres:0000@localhost:5432/new_db"
CHROMA_PATH = "./chroma_db" # Thư mục sẽ chứa database vector

# Load mô hình Qwen3 cục bộ
print("Loading Local Model: Qwen3-Embedding...")
model = SentenceTransformer("Qwen/Qwen3-Embedding-0.6B", trust_remote_code=True)

# Wrapper để LangChain hiểu mô hình Local của bạn
class LocalEmbeddingWrapper:
    def embed_documents(self, texts):
        return model.encode(texts).tolist()
    def embed_query(self, text):
        return model.encode([text])[0].tolist()

def sync_data():
    conn = psycopg2.connect(CONNECTION_STRING)
    cur = conn.cursor()

    # Lấy các bài chưa xử lý từ Postgres
    cur.execute("SELECT id, title, content FROM articles WHERE is_embedded = FALSE")
    rows = cur.fetchall()

    if not rows:
        print("Không có bài mới.")
        return

    text_splitter = RecursiveCharacterTextSplitter(chunk_size=600, chunk_overlap=60)
    embedding_wrapper = LocalEmbeddingWrapper()

    # Khởi tạo (hoặc nạp) ChromaDB từ thư mục cục bộ
    vector_db = Chroma(
        persist_directory=CHROMA_PATH,
        embedding_function=embedding_wrapper
    )

    for row in rows:
        art_id, title, cont = row
        full_text = f"Tiêu đề: {title}. Nội dung: {cont}"
        chunks = text_splitter.split_text(full_text)
        
        # Chuyển thành danh sách Document của LangChain
        docs = [
            Document(
                page_content=chunk,
                metadata={"article_id": art_id, "title": title}
            ) for chunk in chunks
        ]

        # Thêm vào Chroma
        vector_db.add_documents(docs)

        # Đánh dấu đã xong trong Postgres
        cur.execute("UPDATE articles SET is_embedded = TRUE WHERE id = %s", (art_id,))
        conn.commit()
        print(f"✅ Đã đồng bộ bài ID {art_id} vào ChromaDB")

    cur.close()
    conn.close()

# Khởi tạo lại wrapper cho model Qwen
class LocalEmbeddingWrapper:
    def embed_documents(self, texts):
        return model.encode(texts).tolist()
    def embed_query(self, text):
        return model.encode([text])[0].tolist()

def search_news(query_text):
    # 1. Kết nối tới DB đã lưu
    embedding_wrapper = LocalEmbeddingWrapper()
    vector_db = Chroma(
        persist_directory="./chroma_db", 
        embedding_function=embedding_wrapper
    )

    # 2. Thực hiện tìm kiếm (Mặc định Chroma dùng Cosine)
    # k=3 nghĩa là lấy 3 đoạn văn có điểm Cosine cao nhất
    results = vector_db.similarity_search_with_relevance_scores(query_text, k=3)

    for doc, score in results:
        print(f"--- [Độ tương đồng: {score:.4f}] ---")
        print(f"Nội dung: {doc.page_content}")
        print(f"ID bài báo: {doc.metadata['article_id']}")
        print("-" * 30)

    return results

if __name__ == "__main__":
    sync_data()