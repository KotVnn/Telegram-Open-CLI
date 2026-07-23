package telegram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Adapter defines the interface for Telegram bot operations.
type Adapter interface {
	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Message operations
	SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error
	SendMessageWithResult(ctx context.Context, chatID int64, msg OutgoingMessage) (*SendResult, error)
	EditMessage(ctx context.Context, chatID int64, messageID int, text string) error
	SendDocument(ctx context.Context, chatID int64, doc Document) error
	AnswerCallback(ctx context.Context, callbackID string, text string) error

	// Handler registration
	HandleCommand(cmd string, handler HandlerFunc)
	HandleMessage(pattern string, handler HandlerFunc)
	HandleCallback(pattern string, handler CallbackHandlerFunc)
	HandleDefault(handler HandlerFunc)

	// Middleware
	Use(middlewares ...Middleware)
}

// SendResult holds the result of sending a message.
type SendResult struct {
	MessageID int
}

// OutgoingMessage holds data for messages sent by the bot.
type OutgoingMessage struct {
	Text                string
	ParseMode           string
	ReplyMarkup         interface{}
	DisableWebPreview   bool
	DisableNotification bool
}

// Document holds file data for sending.
type Document struct {
	FileID   string
	FileName string
	Content  []byte
	Caption  string
}

// IncomingMessage holds parsed incoming message data.
type IncomingMessage struct {
	MessageID        int
	ChatID           int64
	FromID           int64
	FromUsername      string
	FromFirstName    string
	FromLastName     string
	Text             string
	Command          string
	Args             []string
	ReplyToMessageID int
	Document         *DocumentInfo
}

// DocumentInfo holds metadata about an attached document.
type DocumentInfo struct {
	FileID   string
	FileName string
	MimeType string
	FileSize int64
}

// CallbackQuery holds callback query data from inline buttons.
type CallbackQuery struct {
	ID           string
	ChatID       int64
	FromID       int64
	FromUsername string
	Data         string
	MessageID    int
}

// HandlerFunc handles incoming messages.
type HandlerFunc func(ctx context.Context, msg *IncomingMessage) error

// CallbackHandlerFunc handles callback queries.
type CallbackHandlerFunc func(ctx context.Context, cb *CallbackQuery) error

// Middleware processes messages before/after handlers.
type Middleware func(next HandlerFunc) HandlerFunc

// Bot implements the Adapter interface using go-telegram/bot.
type Bot struct {
	bot               *bot.Bot
	middlewares       []Middleware
	commandHandlers   map[string]HandlerFunc
	messageHandlers   []messageHandler
	callbackHandlers  []callbackHandler
	defaultHandler    HandlerFunc
}

type messageHandler struct {
	pattern string
	handler HandlerFunc
}

type callbackHandler struct {
	pattern string
	handler CallbackHandlerFunc
}

// Config holds Telegram bot configuration.
type Config struct {
	Token         string
	Mode          string
	WebhookURL    string
	WebhookSecret string
}

// New creates a new Bot instance.
func New(cfg Config) (*Bot, error) {
	b := &Bot{
		commandHandlers: make(map[string]HandlerFunc),
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(b.handleUpdate),
	}

	bot, err := bot.New(cfg.Token, opts...)
	if err != nil {
		return nil, err
	}

	b.bot = bot
	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	b.bot.Start(ctx)
	return nil
}

func (b *Bot) Stop(ctx context.Context) error {
	return nil
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error {
	params := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   msg.Text,
	}
	if msg.ParseMode != "" {
		params.ParseMode = models.ParseMode(msg.ParseMode)
	}
	if msg.DisableWebPreview {
		isDisabled := true
		params.LinkPreviewOptions = &models.LinkPreviewOptions{
			IsDisabled: &isDisabled,
		}
	}
	if msg.DisableNotification {
		params.DisableNotification = true
	}
	if msg.ReplyMarkup != nil {
		params.ReplyMarkup = msg.ReplyMarkup
	}

	_, err := b.bot.SendMessage(ctx, params)
	return err
}

func (b *Bot) SendMessageWithResult(ctx context.Context, chatID int64, msg OutgoingMessage) (*SendResult, error) {
	params := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   msg.Text,
	}
	if msg.ParseMode != "" {
		params.ParseMode = models.ParseMode(msg.ParseMode)
	}
	if msg.DisableWebPreview {
		isDisabled := true
		params.LinkPreviewOptions = &models.LinkPreviewOptions{
			IsDisabled: &isDisabled,
		}
	}
	if msg.DisableNotification {
		params.DisableNotification = true
	}
	if msg.ReplyMarkup != nil {
		params.ReplyMarkup = msg.ReplyMarkup
	}

	result, err := b.bot.SendMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	return &SendResult{
		MessageID: result.ID,
	}, nil
}

func (b *Bot) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	params := &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	}
	_, err := b.bot.EditMessageText(ctx, params)
	return err
}

func (b *Bot) SendDocument(ctx context.Context, chatID int64, doc Document) error {
	if doc.FileID != "" {
		params := &bot.SendDocumentParams{
			ChatID:   chatID,
			Document: &models.InputFileString{Data: doc.FileID},
		}
		if doc.Caption != "" {
			params.Caption = doc.Caption
		}
		_, err := b.bot.SendDocument(ctx, params)
		return err
	}

	params := &bot.SendDocumentParams{
		ChatID:   chatID,
		Document: &models.InputFileUpload{
			Filename: doc.FileName,
			Data:     bytes.NewReader(doc.Content),
		},
	}
	if doc.Caption != "" {
		params.Caption = doc.Caption
	}
	_, err := b.bot.SendDocument(ctx, params)
	return err
}

func (b *Bot) AnswerCallback(ctx context.Context, callbackID string, text string) error {
	params := &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	}
	_, err := b.bot.AnswerCallbackQuery(ctx, params)
	return err
}

func (b *Bot) HandleCommand(cmd string, handler HandlerFunc) {
	b.commandHandlers[cmd] = handler
}

func (b *Bot) HandleMessage(pattern string, handler HandlerFunc) {
	b.messageHandlers = append(b.messageHandlers, messageHandler{
		pattern: pattern,
		handler: handler,
	})
}

func (b *Bot) HandleCallback(pattern string, handler CallbackHandlerFunc) {
	b.callbackHandlers = append(b.callbackHandlers, callbackHandler{
		pattern: pattern,
		handler: handler,
	})
}

func (b *Bot) HandleDefault(handler HandlerFunc) {
	b.defaultHandler = handler
}

func (b *Bot) Use(middlewares ...Middleware) {
	b.middlewares = append(b.middlewares, middlewares...)
}

func (b *Bot) handleUpdate(ctx context.Context, _ *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	msg := parseIncomingMessage(update)
	if msg == nil {
		return
	}

	handler := b.resolveHandler(msg)

	chain := handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		chain = b.middlewares[i](chain)
	}

	if err := chain(ctx, msg); err != nil {
		fmt.Fprintf(os.Stderr, "handler error: %v\n", err)
		_ = b.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: "An error occurred. Please try again.",
		})
	}
}

func (b *Bot) handleCallbackQuery(ctx context.Context, cq *models.CallbackQuery) {
	var chatID int64
	var messageID int

	if cq.Message.Type == models.MaybeInaccessibleMessageTypeMessage && cq.Message.Message != nil {
		chatID = cq.Message.Message.Chat.ID
		messageID = cq.Message.Message.ID
	}

	handler := b.resolveCallbackHandler(cq.Data)
	if handler == nil {
		return
	}

	cb := &CallbackQuery{
		ID:           cq.ID,
		ChatID:       chatID,
		FromID:       cq.From.ID,
		FromUsername: cq.From.Username,
		Data:         cq.Data,
		MessageID:    messageID,
	}

	if err := handler(ctx, cb); err != nil {
		fmt.Fprintf(os.Stderr, "callback handler error: %v\n", err)
	}
}

func (b *Bot) resolveCallbackHandler(data string) CallbackHandlerFunc {
	for _, h := range b.callbackHandlers {
		if strings.HasPrefix(data, h.pattern) {
			return h.handler
		}
	}
	return nil
}

func (b *Bot) resolveHandler(msg *IncomingMessage) HandlerFunc {
	if msg.Command != "" {
		if handler, ok := b.commandHandlers[msg.Command]; ok {
			return handler
		}
	}

	for _, h := range b.messageHandlers {
		if strings.Contains(msg.Text, h.pattern) {
			return h.handler
		}
	}

	if b.defaultHandler != nil {
		return b.defaultHandler
	}

	return func(ctx context.Context, msg *IncomingMessage) error {
		return nil
	}
}

func parseIncomingMessage(update *models.Update) *IncomingMessage {
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	result := &IncomingMessage{
		MessageID:     msg.ID,
		ChatID:        msg.Chat.ID,
		FromID:        msg.From.ID,
		FromUsername:   msg.From.Username,
		FromFirstName: msg.From.FirstName,
		FromLastName:  msg.From.LastName,
		Text:          msg.Text,
	}

	if msg.Document != nil {
		result.Document = &DocumentInfo{
			FileID:   msg.Document.FileID,
			FileName: msg.Document.FileName,
			MimeType: msg.Document.MimeType,
			FileSize: int64(msg.Document.FileSize),
		}
	}

	if msg.ReplyToMessage != nil {
		result.ReplyToMessageID = msg.ReplyToMessage.ID
	}

	if len(msg.Entities) > 0 {
		for _, entity := range msg.Entities {
			if entity.Type == "bot_command" {
				command := msg.Text[entity.Offset+1 : entity.Offset+entity.Length]
				result.Command = command
				result.Args = parseArgs(msg.Text[entity.Offset+entity.Length:])
				break
			}
		}
	}

	return result
}

func parseArgs(text string) []string {
	var args []string
	for _, arg := range strings.Fields(text) {
		if arg != "" {
			args = append(args, arg)
		}
	}
	return args
}
