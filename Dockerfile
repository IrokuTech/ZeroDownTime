
# ==========================================
# ETAPA 1 - COMPILACIÓN DE LA APLICACIÓN GO
# ==========================================

FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copiar los archivos necesarios para compilar
COPY go.mod ./
COPY main.go ./

# Compilar la aplicación dentro de Docker
RUN CGO_ENABLED=0 GOOS=linux go build -o servidor .

# ==========================================
# ETAPA 2 - EJECUCIÓN DE LA APLICACIÓN
# ==========================================

FROM alpine:3.22

WORKDIR /app

# Zona horaria de los logs: hora de España (facilita el análisis de los logs)
RUN apk add --no-cache tzdata
ENV TZ=Europe/Madrid

# Copiar el ejecutable generado en la primera etapa
COPY --from=builder /app/servidor ./servidor

# Copiar los recursos necesarios para la web
COPY templates/ ./templates/
COPY static/ ./static/

# Puerto utilizado por el servidor HTTP
EXPOSE 8080

# Ejecutar la aplicación compilada
CMD ["./servidor"]
