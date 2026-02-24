package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

type startChatRequest struct {
	VerticalID string `json:"vertical_id"`
}

type chatMessageRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

func main() {
	engine, err := NewEngine()
	if err != nil {
		log.Fatalf("no se pudo iniciar el motor del chatbot: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/api/verticals", listVerticalsHandler(engine))
	mux.HandleFunc("/api/chat/start", startChatHandler(engine))
	mux.HandleFunc("/api/chat/message", chatMessageHandler(engine))

	addr := ":" + serverPort()
	log.Printf("chatbot core escuchando en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatalf("error de servidor: %v", err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "index.html")
}

func listVerticalsHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "metodo no permitido")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"verticals": engine.ListVerticals(),
		})
	}
}

func startChatHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "metodo no permitido")
			return
		}

		var req startChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "payload invalido")
			return
		}

		response, err := engine.StartSession(strings.TrimSpace(req.VerticalID))
		if err != nil {
			if errors.Is(err, ErrUnknownVertical) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			writeError(w, http.StatusInternalServerError, "error interno al iniciar sesion")
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func chatMessageHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "metodo no permitido")
			return
		}

		var req chatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "payload invalido")
			return
		}

		response, err := engine.HandleMessage(strings.TrimSpace(req.SessionID), strings.TrimSpace(req.Message))
		if err != nil {
			switch {
			case errors.Is(err, ErrUnknownSession):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, ErrEmptyMessage):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "error interno procesando mensaje")
			}

			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func serverPort() string {
	const defaultPort = "8080"

	if value := strings.TrimSpace(os.Getenv("PORT")); value != "" {
		return value
	}

	return defaultPort
}
