package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const puerto = ":8080"

// Datos del entorno de ejecución. En Kubernetes los inyecta el Deployment
// mediante variables de entorno; en local se usan los valores por defecto.
var (
	nombrePod = obtenerNombrePod()
	entorno   = variableOPorDefecto("ENTORNO", "local")
	version   = variableOPorDefecto("APP_VERSION", "dev")
)

// variableOPorDefecto devuelve el valor de una variable de entorno o un valor por defecto.
func variableOPorDefecto(clave, porDefecto string) string {
	if valor := os.Getenv(clave); valor != "" {
		return valor
	}
	return porDefecto
}

// obtenerNombrePod devuelve el nombre del pod (o el hostname del contenedor).
func obtenerNombrePod() string {
	if valor := os.Getenv("POD_NAME"); valor != "" {
		return valor
	}
	nombre, err := os.Hostname()
	if err != nil {
		return "desconocido"
	}
	return nombre
}

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

	datos := map[string]string{
		"Pod":     nombrePod,
		"Entorno": entorno,
		"Version": version,
	}

	if err := plantilla.Execute(w, datos); err != nil {
		log.Printf("Error al renderizar la plantilla: %v", err)
	}
}

// healthHandler comprueba que la aplicación responde.
// Kubernetes lo utiliza en las sondas de disponibilidad (readiness) y de vida (liveness).
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	responderJSON(w, map[string]string{
		"status": "ok",
	})
}

// infoHandler devuelve información básica del proyecto y del pod que responde.
func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	responderJSON(w, map[string]string{
		"nombre":     "ZeroDownTime",
		"tecnologia": "Go + Docker + Kubernetes",
		"entorno":    entorno,
		"version":    version,
		"pod":        nombrePod,
	})
}

// podHandler devuelve en texto plano qué pod atiende la petición.
// Se usa para demostrar el balanceo de carga entre las réplicas.
func podHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responderMetodoNoPermitido(w)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hola desde el pod %s | entorno: %s | versión: %s\n", nombrePod, entorno, version)
}

// responderJSON envía una respuesta HTTP en formato JSON.
func responderJSON(w http.ResponseWriter, datos map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(datos); err != nil {
		log.Printf("Error al generar JSON: %v", err)
	}
}

// registrarPeticiones añade la cabecera X-Pod a todas las respuestas y deja
// constancia de cada petición en el log (útil para el análisis de los logs).
// Las comprobaciones de /health no se registran para no llenar el log con las sondas.
func registrarPeticiones(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		w.Header().Set("X-Pod", nombrePod)
		siguiente.ServeHTTP(w, r)
		if r.URL.Path != "/health" {
			log.Printf("%s %s desde %s (%v)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(inicio))
		}
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", inicioHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/info", infoHandler)
	mux.HandleFunc("/pod", podHandler)

	// Servir los archivos CSS e imágenes desde /static/.
	mux.Handle("/static/", http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("static")),
	))

	servidor := &http.Server{
		Addr:              puerto,
		Handler:           registrarPeticiones(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Apagado ordenado: cuando Kubernetes detiene el pod (SIGTERM) se terminan
	// las peticiones en curso antes de salir, para no cortar ninguna respuesta.
	ctx, detener := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer detener()

	erroresServidor := make(chan error, 1)

	go func() {
		erroresServidor <- servidor.ListenAndServe()
	}()

	log.Printf(
		"Servidor ZeroDownTime iniciado en el puerto %s (pod: %s, entorno: %s, versión: %s)",
		puerto, nombrePod, entorno, version,
	)

	select {
	case <-ctx.Done():
		log.Println("Señal de terminación recibida")
	case err := <-erroresServidor:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Error al iniciar el servidor: %v", err)
		}
		return
	}

	detener()

	ctxCierre, cancelar := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelar()

	if err := servidor.Shutdown(ctxCierre); err != nil {
		log.Printf("Error durante el cierre ordenado: %v", err)
		if err := servidor.Close(); err != nil {
			log.Printf("Error al forzar el cierre: %v", err)
		}
	}

	log.Println("Servidor ZeroDownTime detenido")
}
