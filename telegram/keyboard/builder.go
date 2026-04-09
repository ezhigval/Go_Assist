package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// ReplyBuilder — fluent-конструктор Reply-клавиатуры
type ReplyBuilder struct {
	markup tgbotapi.ReplyKeyboardMarkup
}

// NewReplyKeyboard создаёт новую Reply-клавиатуру
func NewReplyKeyboard(resize bool) *ReplyBuilder {
	return &ReplyBuilder{
		markup: tgbotapi.ReplyKeyboardMarkup{
			ResizeKeyboard:  &resize,
			OneTimeKeyboard: false,
			Keyboard:        [][]tgbotapi.KeyboardButton{},
		},
	}
}

// AddRow добавляет ряд кнопок
func (b *ReplyBuilder) AddRow(btns ...string) *ReplyBuilder {
	row := make([]tgbotapi.KeyboardButton, 0, len(btns))
	for _, txt := range btns {
		row = append(row, tgbotapi.NewKeyboardButton(txt))
	}
	b.markup.Keyboard = append(b.markup.Keyboard, row)
	return b
}

// Build возвращает готовую структуру
func (b *ReplyBuilder) Build() tgbotapi.ReplyKeyboardMarkup {
	return b.markup
}

// InlineBuilder — fluent-конструктор Inline-клавиатуры
type InlineBuilder struct {
	markup tgbotapi.InlineKeyboardMarkup
}

// NewInlineKeyboard создаёт новую Inline-клавиатуру
func NewInlineKeyboard() *InlineBuilder {
	return &InlineBuilder{
		markup: tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}},
	}
}

// AddRow добавляет ряд инлайн-кнопок
func (b *InlineBuilder) AddRow(btns ...tgbotapi.InlineKeyboardButton) *InlineBuilder {
	b.markup.InlineKeyboard = append(b.markup.InlineKeyboard, btns)
	return b
}

// Button создаёт стандартную кнопку
func Button(text, data string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonData(text, data)
}

// URLButton создаёт кнопку-ссылку
func URLButton(text, url string) tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardButtonURL(text, url)
}

// Build возвращает готовую структуру
func (b *InlineBuilder) Build() tgbotapi.InlineKeyboardMarkup {
	return b.markup
}
