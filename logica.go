package main

import (
	"time"
)

const (
	constCantFilasTablero    = 30
	constCantColumnasTablero = 50

	constCantColumnasSerpiente = 2
	constPosX                  = 0
	constPosY                  = 1

	constColumnasDireccion = 2
	constDirX              = 0
	constDirY              = 1

	constSimboloSerpiente = "*"
	constSimboloComida    = "O"
	constSimboloObstaculo = "X"
)

// Vector global para mantener dirección de la cadena
var direccion [constColumnasDireccion]int

// Función para enviar actualizaciones a los clientes
func generarEventos() {
	var (
		tablero   [constCantFilasTablero][constCantColumnasTablero]string
		serpiente [][constCantColumnasSerpiente]int
	)

	// Se genera tablero por primera vez
	tablero = generarTablero()

	// Se genera cadena (posición inicial) por primera vez
	serpiente = generarSerpiente()

	// Se genera la primera comida
	generarComida(&tablero)

	// Se actualiza el tablero con la serpiente por primera vez
	actualizarSerpienteEnTablero(&tablero, serpiente)

	for {
		// Se actualizan las posiciones de la serpiente según la dirección
		calcularNuevaPosicionSerpiente(serpiente)

		// Si perdió, se devuelve pantalla gameOver
		if verificarObstaculos(tablero, serpiente) {
			enviarGameOver()

			return
		}

		// Se actualiza serpiente si es que se comió algún alimento
		serpiente = verificarNuevoSegmento(&tablero, serpiente)

		// Se actualiza el tablero con los valores de serpiente
		actualizarSerpienteEnTablero(&tablero, serpiente)

		// Se envía actualización de tablero al cliente para mostrar en pantalla
		enviarActualizacionTablero(tablero)

		// Espera un tiempo antes de generar una nuevo evento
		time.Sleep(100 * time.Millisecond)
	}
}

func generarTablero() [constCantFilasTablero][constCantColumnasTablero]string {

	var tablero [constCantFilasTablero][constCantColumnasTablero]string

	for t := 0; t < constCantFilasTablero; t++ {
		for v := 0; v < constCantColumnasTablero; v++ {

			if v == 0 {

				tablero[t][v] = constSimboloObstaculo

			} else if t == 0 {

				tablero[t][v] = constSimboloObstaculo

			}

			tablero[constCantFilasTablero-1][v] = constSimboloObstaculo

			tablero[t][constCantColumnasTablero-1] = constSimboloObstaculo

		}
	}

	return tablero
}

func generarSerpiente() [][constCantColumnasSerpiente]int {
	var (
		serpiente [][constCantColumnasSerpiente]int
	)

	// Programar aquí

	return serpiente
}

func actualizarSerpienteEnTablero(tablero *[constCantFilasTablero][constCantColumnasTablero]string,
	serpiente [][constCantColumnasSerpiente]int) {
	// Programar aquí
}

func calcularNuevaPosicionSerpiente(serpiente [][constCantColumnasSerpiente]int) {
	// Programar aquí
}

func verificarObstaculos(tablero [constCantFilasTablero][constCantColumnasTablero]string,
	serpiente [][constCantColumnasSerpiente]int) bool {
	var resultado bool

	// Programar aquí

	return resultado
}

func verificarNuevoSegmento(tablero *[constCantFilasTablero][constCantColumnasTablero]string,
	serpiente [][constCantColumnasSerpiente]int) [][constCantColumnasSerpiente]int {
	// Programar aquí

	return serpiente
}

func generarComida(tablero *[constCantFilasTablero][constCantColumnasTablero]string) {
	// Programar aquí
}
