package message

import "strings"

// ExtractCallbackData парсит callback_data в формате key:value
func ExtractCallbackData(data string) (key string, value string, ok bool) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ParseCommandArgs разделяет команду и аргументы: "/calc 10 + 5" → ["calc", "10 + 5"]
func ParseCommandArgs(text string) (cmd string, args string) {
	parts := strings.SplitN(strings.TrimPrefix(text, "/"), " ", 2)
	cmd = strings.ToLower(parts[0])
	if len(parts) > 1 {
		args = parts[1]
	}
	return cmd, args
}
