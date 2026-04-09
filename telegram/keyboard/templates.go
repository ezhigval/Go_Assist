package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainReplyKeyboard — шаблон главного меню
// STUB: Dynamic keyboards require MenuProvider(ctx, userID) instead of static [][]; avoid hardcoding button texts in multiple places.
func MainReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return NewReplyKeyboard(true).
		AddRow("📦 Каталог", "🧮 Калькулятор").
		AddRow("📊 Статистика", "⚙️ Настройки").
		Build()
}

// CancelInlineKeyboard — кнопка отмены для диалогов
func CancelInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return NewInlineKeyboard().
		AddRow(Button("❌ Отмена", "cancel_dialog")).
		Build()
}
