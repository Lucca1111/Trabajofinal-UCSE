package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	ErrUnknownVertical = errors.New("vertical no encontrada")
	ErrUnknownSession  = errors.New("sesion no encontrada")
	ErrEmptyMessage    = errors.New("mensaje vacio")
)

//go:embed verticals/*.json
var embeddedVerticals embed.FS

type VerticalConfig struct {
	ID                string      `json:"id"`
	DisplayName       string      `json:"display_name"`
	WelcomeMessage    string      `json:"welcome_message"`
	CompletionMessage string      `json:"completion_message"`
	HandoffMessage    string      `json:"handoff_message"`
	FallbackMessage   string      `json:"fallback_message"`
	HandoffKeywords   []string    `json:"handoff_keywords"`
	FAQs              []FAQ       `json:"faqs"`
	Fields            []LeadField `json:"fields"`
}

type LeadField struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	Question        string `json:"question"`
	Required        bool   `json:"required"`
	ValidationRegex string `json:"validation_regex"`
	Example         string `json:"example"`
}

type FAQ struct {
	Keywords []string `json:"keywords"`
	Answer   string   `json:"answer"`
}

type Session struct {
	ID           string            `json:"id"`
	VerticalID   string            `json:"vertical_id"`
	Collected    map[string]string `json:"collected"`
	CurrentField int               `json:"current_field"`
	Completed    bool              `json:"completed"`
}

type VerticalSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type ChatResponse struct {
	SessionID    string            `json:"session_id"`
	VerticalID   string            `json:"vertical_id"`
	Reply        string            `json:"reply"`
	Completed    bool              `json:"completed"`
	MissingField string            `json:"missing_field,omitempty"`
	Collected    map[string]string `json:"collected,omitempty"`
}

type Engine struct {
	mu        sync.Mutex
	verticals map[string]VerticalConfig
	sessions  map[string]*Session
}

func NewEngine() (*Engine, error) {
	verticals, err := loadVerticalsFromEmbed()
	if err != nil {
		return nil, err
	}

	return &Engine{
		verticals: verticals,
		sessions:  make(map[string]*Session),
	}, nil
}

func (e *Engine) ListVerticals() []VerticalSummary {
	e.mu.Lock()
	defer e.mu.Unlock()

	summaries := make([]VerticalSummary, 0, len(e.verticals))
	for _, config := range e.verticals {
		summaries = append(summaries, VerticalSummary{
			ID:          config.ID,
			DisplayName: config.DisplayName,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].DisplayName < summaries[j].DisplayName
	})

	return summaries
}

func (e *Engine) StartSession(verticalID string) (ChatResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if verticalID == "" {
		verticalID = e.firstVerticalID()
	}

	config, ok := e.verticals[verticalID]
	if !ok {
		return ChatResponse{}, fmt.Errorf("%w: %s", ErrUnknownVertical, verticalID)
	}

	sessionID, err := randomID()
	if err != nil {
		return ChatResponse{}, err
	}

	session := &Session{
		ID:           sessionID,
		VerticalID:   config.ID,
		Collected:    make(map[string]string),
		CurrentField: 0,
		Completed:    false,
	}

	e.sessions[sessionID] = session

	reply := config.WelcomeMessage
	if nextField, exists := nextPendingField(config.Fields, session.CurrentField, session.Collected); exists {
		reply = fmt.Sprintf("%s\n\n%s", config.WelcomeMessage, nextField.Question)
	}

	return ChatResponse{
		SessionID:    session.ID,
		VerticalID:   session.VerticalID,
		Reply:        reply,
		Completed:    false,
		MissingField: currentMissingField(config, session),
		Collected:    copyCollected(session.Collected),
	}, nil
}

func (e *Engine) HandleMessage(sessionID, message string) (ChatResponse, error) {
	if strings.TrimSpace(message) == "" {
		return ChatResponse{}, ErrEmptyMessage
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	session, ok := e.sessions[sessionID]
	if !ok {
		return ChatResponse{}, fmt.Errorf("%w: %s", ErrUnknownSession, sessionID)
	}

	config, ok := e.verticals[session.VerticalID]
	if !ok {
		return ChatResponse{}, fmt.Errorf("%w: %s", ErrUnknownVertical, session.VerticalID)
	}

	if session.Completed {
		return ChatResponse{
			SessionID:  session.ID,
			VerticalID: session.VerticalID,
			Reply:      "Esta sesion ya fue completada. Puedes iniciar una nueva para otro lead.",
			Completed:  true,
			Collected:  copyCollected(session.Collected),
		}, nil
	}

	lowerMessage := strings.ToLower(message)

	if containsAny(lowerMessage, config.HandoffKeywords) {
		session.Completed = true
		reply := fmt.Sprintf("%s\n\n%s", config.HandoffMessage, summaryFromFields(config.Fields, session.Collected))
		return ChatResponse{
			SessionID:  session.ID,
			VerticalID: session.VerticalID,
			Reply:      strings.TrimSpace(reply),
			Completed:  true,
			Collected:  copyCollected(session.Collected),
		}, nil
	}

	if faqAnswer, isFAQ := searchFAQ(lowerMessage, config.FAQs); isFAQ {
		reply := faqAnswer
		if field, exists := nextPendingField(config.Fields, session.CurrentField, session.Collected); exists {
			reply = fmt.Sprintf("%s\n\nPara continuar: %s", faqAnswer, field.Question)
		}

		return ChatResponse{
			SessionID:    session.ID,
			VerticalID:   session.VerticalID,
			Reply:        reply,
			Completed:    false,
			MissingField: currentMissingField(config, session),
			Collected:    copyCollected(session.Collected),
		}, nil
	}

	field, index, exists := pendingFieldWithIndex(config.Fields, session.CurrentField, session.Collected)
	if !exists {
		session.Completed = true

		return ChatResponse{
			SessionID:  session.ID,
			VerticalID: session.VerticalID,
			Reply:      buildCompletionReply(config, session.Collected),
			Completed:  true,
			Collected:  copyCollected(session.Collected),
		}, nil
	}

	if !field.Required && isSkipIntent(lowerMessage) {
		session.CurrentField = index + 1
		return e.replyWithNextQuestionOrCompletion(session, config)
	}

	value, valid := extractFieldValue(field, message)
	if !valid {
		reply := config.FallbackMessage
		if strings.TrimSpace(reply) == "" {
			reply = "No pude validar ese dato."
		}

		if strings.TrimSpace(field.Example) != "" {
			reply = fmt.Sprintf("%s Ejemplo valido: %s", reply, field.Example)
		}

		reply = fmt.Sprintf("%s\n\n%s", strings.TrimSpace(reply), field.Question)

		return ChatResponse{
			SessionID:    session.ID,
			VerticalID:   session.VerticalID,
			Reply:        reply,
			Completed:    false,
			MissingField: field.Key,
			Collected:    copyCollected(session.Collected),
		}, nil
	}

	session.Collected[field.Key] = value
	session.CurrentField = index + 1

	return e.replyWithNextQuestionOrCompletion(session, config)
}

func (e *Engine) replyWithNextQuestionOrCompletion(session *Session, config VerticalConfig) (ChatResponse, error) {
	nextField, exists := nextPendingField(config.Fields, session.CurrentField, session.Collected)
	if exists {
		return ChatResponse{
			SessionID:    session.ID,
			VerticalID:   session.VerticalID,
			Reply:        nextField.Question,
			Completed:    false,
			MissingField: nextField.Key,
			Collected:    copyCollected(session.Collected),
		}, nil
	}

	session.Completed = true

	return ChatResponse{
		SessionID:  session.ID,
		VerticalID: session.VerticalID,
		Reply:      buildCompletionReply(config, session.Collected),
		Completed:  true,
		Collected:  copyCollected(session.Collected),
	}, nil
}

func buildCompletionReply(config VerticalConfig, collected map[string]string) string {
	reply := strings.TrimSpace(config.CompletionMessage)
	if reply == "" {
		reply = "Perfecto, ya tengo los datos necesarios."
	}

	summary := summaryFromFields(config.Fields, collected)
	if summary != "" {
		reply = fmt.Sprintf("%s\n\n%s", reply, summary)
	}

	if strings.TrimSpace(config.HandoffMessage) != "" {
		reply = fmt.Sprintf("%s\n\n%s", reply, strings.TrimSpace(config.HandoffMessage))
	}

	return strings.TrimSpace(reply)
}

func summaryFromFields(fields []LeadField, collected map[string]string) string {
	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := collected[field.Key]
		if !ok {
			continue
		}

		lines = append(lines, fmt.Sprintf("- %s: %s", field.Label, value))
	}

	if len(lines) == 0 {
		return ""
	}

	return "Resumen de lead:\n" + strings.Join(lines, "\n")
}

func currentMissingField(config VerticalConfig, session *Session) string {
	field, exists := nextPendingField(config.Fields, session.CurrentField, session.Collected)
	if !exists {
		return ""
	}

	return field.Key
}

func pendingFieldWithIndex(fields []LeadField, startIndex int, collected map[string]string) (LeadField, int, bool) {
	for i := startIndex; i < len(fields); i++ {
		field := fields[i]
		if _, exists := collected[field.Key]; exists {
			continue
		}

		return field, i, true
	}

	return LeadField{}, -1, false
}

func nextPendingField(fields []LeadField, startIndex int, collected map[string]string) (LeadField, bool) {
	field, _, exists := pendingFieldWithIndex(fields, startIndex, collected)
	return field, exists
}

func searchFAQ(message string, faqs []FAQ) (string, bool) {
	for _, faq := range faqs {
		for _, keyword := range faq.Keywords {
			if strings.Contains(message, strings.ToLower(keyword)) {
				return faq.Answer, true
			}
		}
	}

	return "", false
}

func containsAny(message string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func isSkipIntent(message string) bool {
	skipKeywords := []string{"omitir", "saltar", "paso", "prefiero no responder", "skip"}
	return containsAny(message, skipKeywords)
}

func extractFieldValue(field LeadField, raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	if field.ValidationRegex == "" {
		return value, true
	}

	re, err := regexp.Compile(field.ValidationRegex)
	if err != nil {
		return "", false
	}

	if !re.MatchString(value) {
		return "", false
	}

	return value, true
}

func loadVerticalsFromEmbed() (map[string]VerticalConfig, error) {
	entries, err := fs.ReadDir(embeddedVerticals, "verticals")
	if err != nil {
		return nil, err
	}

	verticals := make(map[string]VerticalConfig, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		raw, err := fs.ReadFile(embeddedVerticals, "verticals/"+entry.Name())
		if err != nil {
			return nil, err
		}

		var cfg VerticalConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("error parseando vertical %s: %w", entry.Name(), err)
		}

		if strings.TrimSpace(cfg.ID) == "" {
			return nil, fmt.Errorf("vertical %s sin id", entry.Name())
		}

		if len(cfg.Fields) == 0 {
			return nil, fmt.Errorf("vertical %s sin campos de captura", cfg.ID)
		}

		for _, field := range cfg.Fields {
			if strings.TrimSpace(field.Key) == "" || strings.TrimSpace(field.Question) == "" {
				return nil, fmt.Errorf("vertical %s contiene campos invalidos", cfg.ID)
			}

			if strings.TrimSpace(field.ValidationRegex) != "" {
				if _, err := regexp.Compile(field.ValidationRegex); err != nil {
					return nil, fmt.Errorf("regex invalida en vertical %s campo %s: %w", cfg.ID, field.Key, err)
				}
			}
		}

		verticals[cfg.ID] = cfg
	}

	if len(verticals) == 0 {
		return nil, errors.New("no se cargaron verticales")
	}

	return verticals, nil
}

func randomID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func copyCollected(collected map[string]string) map[string]string {
	result := make(map[string]string, len(collected))
	for key, value := range collected {
		result[key] = value
	}

	return result
}

func (e *Engine) firstVerticalID() string {
	ids := make([]string, 0, len(e.verticals))
	for id := range e.verticals {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}

	return ids[0]
}
