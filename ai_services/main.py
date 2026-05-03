from fastapi import FastAPI
from pydantic import BaseModel
from langchain_chroma import Chroma
from sentence_transformers import SentenceTransformer
from bs4 import BeautifulSoup
import os

app = FastAPI()

# --- KHU VỰC TIỆN ÍCH ---
def clean_html(html_text):
    if not html_text or html_text == "None":
        return ""
    soup = BeautifulSoup(html_text, "html.parser")
    return soup.get_text(separator=' ').strip()

class ChatQuery(BaseModel):
    message: str

# --- KHU VỰC AI MODEL ---
print("Loading Model...")
model = SentenceTransformer("Qwen/Qwen3-Embedding-0.6B", trust_remote_code=True)

class LocalEmbeddingWrapper:
    def embed_documents(self, texts): return model.encode(texts).tolist()
    def embed_query(self, text): return model.encode([text])[0].tolist()

vector_db = Chroma(
    persist_directory="./chroma_db",
    embedding_function=LocalEmbeddingWrapper()
)

# --- ENDPOINT CHÍNH ---
@app.post("/chat")
async def chat(query: ChatQuery):
    user_msg = query.message
    # Tìm 3 đoạn văn liên quan
    docs = vector_db.similarity_search(user_msg, k=3)
    
    # Lấy danh sách ID duy nhất
    article_ids = list(set([d.metadata.get("article_id") for d in docs]))
    
    # Trả về danh sách ID cho Go
    return {"article_ids": article_ids}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)