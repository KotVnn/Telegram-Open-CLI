package telegram

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// InlineKeyboard represents an inline keyboard markup.
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton
}

// InlineKeyboardButton represents an inline keyboard button.
type InlineKeyboardButton struct {
	Text         string
	CallbackData string
}

// ReplyKeyboard represents a reply keyboard markup.
type ReplyKeyboard struct {
	Keyboard        [][]KeyboardButton
	ResizeKeyboard  bool
	OneTimeKeyboard bool
}

// KeyboardButton represents a reply keyboard button.
type KeyboardButton struct {
	Text string
}

// NewSessionKeyboard creates a keyboard for session selection.
func NewSessionKeyboard(sessions []struct{ ID, Title string }) InlineKeyboard {
	var rows [][]InlineKeyboardButton

	for _, session := range sessions {
		rows = append(rows, []InlineKeyboardButton{
			{
				Text:         session.Title,
				CallbackData: fmt.Sprintf("session:%s", session.ID),
			},
		})
	}

	rows = append(rows, []InlineKeyboardButton{
		{
			Text:         "+ New Session",
			CallbackData: "new_session",
		},
	})

	return InlineKeyboard{InlineKeyboard: rows}
}

// NewConfirmKeyboard creates a confirmation keyboard.
func NewConfirmKeyboard(action string) InlineKeyboard {
	return InlineKeyboard{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Yes", CallbackData: fmt.Sprintf("confirm:%s", action)},
				{Text: "No", CallbackData: "cancel"},
			},
		},
	}
}

// NewModelKeyboard creates a model selection keyboard.
func NewModelKeyboard(models []string) InlineKeyboard {
	var rows [][]InlineKeyboardButton
	for _, model := range models {
		rows = append(rows, []InlineKeyboardButton{
			{
				Text:         model,
				CallbackData: fmt.Sprintf("model:%s", model),
			},
		})
	}
	return InlineKeyboard{InlineKeyboard: rows}
}

// NewAgentKeyboard creates an agent selection keyboard.
func NewAgentKeyboard(agents []string) InlineKeyboard {
	var rows [][]InlineKeyboardButton
	for _, agent := range agents {
		rows = append(rows, []InlineKeyboardButton{
			{
				Text:         agent,
				CallbackData: fmt.Sprintf("agent:%s", agent),
			},
		})
	}
	return InlineKeyboard{InlineKeyboard: rows}
}

// KeyboardBuilder provides a fluent API for building keyboards.
type KeyboardBuilder struct {
	rows [][]InlineKeyboardButton
}

// NewKeyboard creates a new KeyboardBuilder.
func NewKeyboard() *KeyboardBuilder {
	return &KeyboardBuilder{}
}

// AddButton adds a button to the current row.
func (b *KeyboardBuilder) AddButton(text, data string) *KeyboardBuilder {
	if len(b.rows) == 0 {
		b.rows = append(b.rows, []InlineKeyboardButton{})
	}
	b.rows[len(b.rows)-1] = append(b.rows[len(b.rows)-1], InlineKeyboardButton{
		Text:         text,
		CallbackData: data,
	})
	return b
}

// NewRow starts a new row.
func (b *KeyboardBuilder) NewRow() *KeyboardBuilder {
	b.rows = append(b.rows, []InlineKeyboardButton{})
	return b
}

// Build builds the keyboard.
func (b *KeyboardBuilder) Build() InlineKeyboard {
	return InlineKeyboard{InlineKeyboard: b.rows}
}

// ToTelegram converts to go-telegram InlineKeyboardMarkup.
func (k InlineKeyboard) ToTelegram() models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, len(k.InlineKeyboard))
	for i, row := range k.InlineKeyboard {
		buttons := make([]models.InlineKeyboardButton, len(row))
		for j, btn := range row {
			buttons[j] = models.InlineKeyboardButton{
				Text:         btn.Text,
				CallbackData: btn.CallbackData,
			}
		}
		rows[i] = buttons
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
