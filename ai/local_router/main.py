from fastapi import FastAPI

app = FastAPI(title="Modulr Local AI Router")


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/infer")
def infer(payload: dict):
    text = str(payload.get("text", "")).strip()
    if len(text) < 10:
        return {"action": "", "confidence": 0.35}
    return {"action": "link_todo_to_calendar", "confidence": 0.78}
