from fastapi import FastAPI

app = FastAPI(title="Modulr Local AI Router")


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/infer")
def infer(payload: dict):
    text = str(payload.get("text", "")).strip()
    scope = str(payload.get("scope") or "personal")

    if len(text) < 10:
        return {"decisions": []}

    text_lower = text.lower()
    if "напомин" in text_lower or "deadline" in text_lower:
        return {
            "decisions": [
                {
                    "target": "tracker",
                    "action": "create_reminder",
                    "confidence": 0.88,
                    "scope": scope,
                    "parameters": {"note": text},
                }
            ]
        }
    if "встреч" in text_lower or "созвон" in text_lower or "календар" in text_lower:
        return {
            "decisions": [
                {
                    "target": "calendar",
                    "action": "create_event",
                    "confidence": 0.9,
                    "scope": scope,
                    "parameters": {"title": text},
                }
            ]
        }
    if "бюджет" in text_lower or "трат" in text_lower or "расход" in text_lower:
        return {
            "decisions": [
                {
                    "target": "finance",
                    "action": "create_transaction",
                    "confidence": 0.84,
                    "scope": scope,
                    "parameters": {"memo": text},
                }
            ]
        }

    return {
        "decisions": [
            {
                "target": "knowledge",
                "action": "save_query",
                "confidence": 0.74,
                "scope": scope,
                "parameters": {"text": text},
            }
        ]
    }
