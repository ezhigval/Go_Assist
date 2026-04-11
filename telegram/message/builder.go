package message

import (
	"fmt"
	"telegram/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BuildMessageConfig преобразует Response в конфигурацию отправки Telegram
func BuildMessageConfig(chatID int64, resp *handler.Response) (tgbotapi.Chattable, error) {
	if resp.Delete {
		msg := tgbotapi.NewDeleteMessage(chatID, 0) // msgID должен передаваться отдельно в продакшене
		return msg, nil
	}

	if resp.Edit {
		msg := tgbotapi.NewEditMessageText(chatID, 0, resp.Text)
		msg.ParseMode = resp.ParseMode
		if kb, ok := resp.Keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = &kb
		}
		return msg, nil
	}

	msg := tgbotapi.NewMessage(chatID, resp.Text)
	msg.ParseMode = resp.ParseMode
	if resp.Keyboard != nil {
		msg.ReplyMarkup = resp.Keyboard
	}
	return msg, nil
}

// FormatError стандартно форматирует ошибки для отправки
func FormatError(err error) string {
	return fmt.Sprintf("⚠️ Произошла ошибка: `%s`\nПопробуйте позже или обратитесь в поддержку.", err.Error())
}
