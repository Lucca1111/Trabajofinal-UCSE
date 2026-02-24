# Chatbot Core Multi-Rubro (MVP)

Aplicacion base para vender chatbots por vertical:

- **Vertical inicial:** Inmobiliaria
- **Vertical ejemplo extra:** Clinica Dental
- **Stack:** Go + API REST + frontend HTML/JS
- **Objetivo:** reutilizar el mismo motor de conversacion y solo cambiar configuraciones.

## Que incluye este MVP

1. Motor de conversacion con:
   - sesiones por usuario
   - captura de lead por campos
   - validacion por regex
   - FAQs por vertical
   - handoff a humano
2. Configuracion por vertical en JSON (`/verticals/*.json`)
3. UI web para probar conversaciones (`index.html`)
4. Tests unitarios (`engine_test.go`)

## Ejecutar local

```bash
go run .
```

Abrir:

```text
http://localhost:8080
```

## Endpoints

### GET `/api/verticals`
Devuelve verticales disponibles.

### POST `/api/chat/start`
Body:

```json
{
  "vertical_id": "real_estate"
}
```

### POST `/api/chat/message`
Body:

```json
{
  "session_id": "abc123",
  "message": "Quiero comprar"
}
```

## Como agregar otro rubro

1. Copiar un archivo de `verticals/`.
2. Cambiar mensajes, FAQs y campos.
3. Ajustar regex de validacion de cada campo.
4. Reiniciar el servidor.

No hace falta tocar el motor para casos estandar.

## Crear otro repositorio en GitHub

Si quieres separar este MVP en un repo nuevo:

```bash
gh repo create chatbot-core-multirubro --private --source=. --remote=core --push
```

Luego podras seguir empujando a ese remoto:

```bash
git push -u core cursor/reconocimiento-plan-pro-af1c
```
