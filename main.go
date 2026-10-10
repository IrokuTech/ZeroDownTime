package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

const puerto = ":8080"

// responderMetodoNoPermitido devuelve HTTP 405 cuando el método no está admitido.
func responderMetodoNoPermitido(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET")
	http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
}

// inicioHandler muestra la página principal.
func inicioHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	plantilla, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Printf("Error al cargar la plantilla: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	if err := plantilla.Execute(w, nil); err != nil {
		log.Printf("Error al renderizar la plantilla: %v", err)
	}
}

// healthHandler comprueba que la aplicación responde.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	responderJSON(w, map[string]string{
		"status": "ok",
	})
}

// infoHandler devuelve información básica del proyecto.
func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	responderJSON(w, map[string]string{
		"nombre":   "ZeroDownTime",
		"producto": "Producto 1",
		"entorno":  "Go + Docker",
	})
}

// responderJSON envía una respuesta HTTP en formato JSON.
func responderJSON(w http.ResponseWriter, datos map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(datos); err != nil {
		log.Printf("Error al generar JSON: %v", err)
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", inicioHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/info", infoHandler)

	// Servir los archivos CSS e imágenes desde /static/.
	mux.Handle("/static/", http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("static")),
	))

	log.Printf("Servidor ZeroDownTime iniciado en http://localhost%s", puerto)

	if err := http.ListenAndServe(puerto, mux); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
