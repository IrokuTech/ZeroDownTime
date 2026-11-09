# ZeroDownTime

Proyecto grupal de la asignatura DevOps y Cloud Computing.

## Producto 1 — Go + Docker

Preparación del entorno de desarrollo, construcción de una aplicación web con Go y contenerización mediante Docker.

## Producto 2 — Despliegue en producción

Despliegue automático en Kubernetes (Minikube) sobre una instancia EC2 de AWS mediante Jenkins.

| Rama | Entorno (namespace) | NodePort |
|---|---|---|
| `develop` | `pruebas` | 30081 |
| `main` | `produccion` | 30080 |

Flujo: `git push` → webhook de GitHub → Jenkins (Multibranch Pipeline) → imagen en Docker Hub → `kubectl apply` → 2 réplicas con actualización progresiva sin cortes.

- `Jenkinsfile`: pipeline de construcción, publicación, despliegue y verificación.
- `k8s/`: manifiestos del Deployment (2 réplicas, sondas, RollingUpdate) y del Service (NodePort).
- Rutas de la aplicación: `/`, `/health`, `/info` y `/pod` (identifica el pod que responde).

## Equipo

- Andrey Carrera Mosquera
- David Cuenca Guirado
- Yile Ma Ye