package main

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadVerticalsIncludesRealEstate(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	verticals := engine.ListVerticals()
	if len(verticals) < 2 {
		t.Fatalf("se esperaban al menos 2 verticales, obtenido %d", len(verticals))
	}

	found := false
	for _, vertical := range verticals {
		if vertical.ID == "real_estate" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("la vertical real_estate no fue cargada")
	}
}

func TestRealEstateLeadFlowCompletes(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	start, err := engine.StartSession("real_estate")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	if start.SessionID == "" {
		t.Fatal("se esperaba un session_id generado")
	}

	if !strings.Contains(strings.ToLower(start.Reply), "nombre") {
		t.Fatalf("respuesta inicial inesperada: %q", start.Reply)
	}

	testMessages := []string{
		"Juan Perez",
		"compra",
		"departamento",
		"Nueva Cordoba",
		"120000 USD",
		"omitir",
		"+54 9 351 555 1234",
	}

	var response ChatResponse
	for _, userMessage := range testMessages {
		response, err = engine.HandleMessage(start.SessionID, userMessage)
		if err != nil {
			t.Fatalf("HandleMessage(%q) error = %v", userMessage, err)
		}
	}

	if !response.Completed {
		t.Fatalf("la sesion deberia estar completada, respuesta final: %+v", response)
	}

	if response.Collected["operation_type"] != "compra" {
		t.Fatalf("operation_type incorrecto: %#v", response.Collected["operation_type"])
	}

	if response.Collected["contact_phone"] == "" {
		t.Fatal("contact_phone no fue almacenado")
	}
}

func TestUnknownSession(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	_, err = engine.HandleMessage("inexistente", "hola")
	if err == nil {
		t.Fatal("se esperaba error en sesion inexistente")
	}

	if !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestFAQDoesNotBreakFlow(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	start, err := engine.StartSession("real_estate")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	response, err := engine.HandleMessage(start.SessionID, "Cual es la comision?")
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if response.Completed {
		t.Fatal("una FAQ no deberia completar la sesion")
	}

	if !strings.Contains(strings.ToLower(response.Reply), "comision") {
		t.Fatalf("respuesta FAQ inesperada: %q", response.Reply)
	}

	if response.MissingField != "full_name" {
		t.Fatalf("se esperaba que falte full_name, se obtuvo %q", response.MissingField)
	}
}
