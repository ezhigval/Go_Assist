package ai

import "regexp"

var (
	emailPattern = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	phonePattern = regexp.MustCompile(`\+?\d[\d\-\s()]{7,}\d`)
	cardPattern  = regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`)
)

// RedactPII скрывает базовые PII-паттерны перед передачей текста во внешние LLM.
func RedactPII(text string) string {
	out := emailPattern.ReplaceAllString(text, "<redacted:email>")
	out = phonePattern.ReplaceAllString(out, "<redacted:phone>")
	out = cardPattern.ReplaceAllString(out, "<redacted:card>")
	return out
}
