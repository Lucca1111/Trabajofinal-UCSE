package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// Agrega un canal para enviar actualizaciones a los clientes
var updates = make(chan string)

func main() {
	rand.Seed(time.Now().Unix())

	// Ruta para servir la página principal
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Ruta para manejar la solicitud POST del evento de teclado
	http.HandleFunc("/keypress", keyPressHandler)

	// Agrega una nueva ruta para SSE
	http.HandleFunc("/updates", updatesHandler)

	// Agrega una nueva ruta para "game over"
	http.HandleFunc("/gameover", gameoverHandler)

	// Inicia una goroutine para enviar actualizaciones a los clientes
	go generarEventos()

	fmt.Println("Abrir navegador e ingresar a http://localhost:8080/")

	// Inicia el servidor en el puerto 8080
	http.ListenAndServe(":8080", nil)
}

// Handler para "game over"
func gameoverHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "gameover.html")
}

// Handler para la conexión SSE
func updatesHandler(w http.ResponseWriter, r *http.Request) {
	// Establece las cabeceras para indicar que esta es una conexión SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Mantén la conexión abierta
	for {
		// Espera a recibir una actualización desde el canal de actualizaciones
		update := <-updates

		// Escribe la actualización al cliente
		fmt.Fprintf(w, "data: %s\n\n", update)

		// Flushea el buffer para enviar la actualización inmediatamente
		w.(http.Flusher).Flush()
	}
}

// Handler para manejar la solicitud POST cuando el usuario presiona una tecla
func keyPressHandler(w http.ResponseWriter, r *http.Request) {
	// Leer los datos JSON enviados en la solicitud
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Error al leer los datos JSON", http.StatusBadRequest)
		return
	}

	// Obtener la tecla presionada del mapa de datos
	keyPressed, ok := data["key"]
	if !ok {
		http.Error(w, "Tecla no proporcionada", http.StatusBadRequest)
		return
	}

	// Actualizar la dirección según la tecla presionada
	switch keyPressed {
	case "ArrowRight":
		{
			direccion[constDirY] = 1
			direccion[constDirX] = 0
		}
	case "ArrowLeft":
		{
			direccion[constDirY] = -1
			direccion[constDirX] = 0
		}
	case "ArrowUp":
		{
			direccion[constDirY] = 0
			direccion[constDirX] = -1
		}
	case "ArrowDown":
		{
			direccion[constDirY] = 0
			direccion[constDirX] = 1
		}
	}

	// Responder con un mensaje de éxito
	w.WriteHeader(http.StatusOK)
}

// Se envía actualización de tablero al cliente
func enviarActualizacionTablero(tablero [constCantFilasTablero][constCantColumnasTablero]string) {
	// Convierte la matriz en una cadena JSON
	update, err := json.Marshal(tablero)
	if err != nil {
		fmt.Println("Error al convertir la matriz en JSON:", err)
	}

	// Envía la actualización a todos los clientes conectados
	updates <- string(update)
}

func enviarGameOver() {
	// Envía la actualización de game over a todos los clientes conectados
	updates <- "{\"game_over\": true}"

	fmt.Println("Game Over")
}
