// Pipeline de ZeroDownTime (Multibranch Pipeline)
//
// Ramas y entornos:
//   develop -> namespace "pruebas"     (NodePort 30081)
//   main    -> namespace "produccion"  (NodePort 30080)
//
// Credenciales necesarias en Jenkins:
//   dockerhub -> Username with password
//                Usuario Docker Hub + Personal Access Token
//
// Requisitos del agente Jenkins:
//   Docker, kubectl y acceso al clúster Minikube.

pipeline {
    agent any

    options {
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '15'))
        timestamps()
    }

    environment {
        DOCKERHUB_USER = 'irokutech'
        APP            = 'zerodowntime'
    }

    stages {
        stage('Preparar') {
            steps {
                script {
                    def destinos = [
                        'main'   : [ns: 'produccion', nodePort: '30080'],
                        'develop': [ns: 'pruebas',    nodePort: '30081']
                    ]

                    def destino = destinos[env.BRANCH_NAME]

                    if (destino == null) {
                        error("La rama '${env.BRANCH_NAME}' no tiene un entorno de despliegue asignado")
                    }

                    env.NAMESPACE = destino.ns
                    env.NODE_PORT = destino.nodePort

                    env.GIT_CORTO = sh(
                        script: 'git rev-parse --short HEAD',
                        returnStdout: true
                    ).trim()

                    env.VERSION = "${env.BRANCH_NAME}-${env.BUILD_NUMBER}-${env.GIT_CORTO}"
                    env.IMAGEN  = "${env.DOCKERHUB_USER}/${env.APP}:${env.VERSION}"

                    currentBuild.displayName = "#${env.BUILD_NUMBER} → ${env.NAMESPACE}"
                    currentBuild.description = "Imagen: ${env.IMAGEN}"
                }

                sh 'git log -1 --pretty=format:"Commit %h de %an: %s%n"'
            }
        }

        stage('Construir imagen') {
            steps {
                sh 'docker build -t "$IMAGEN" .'
            }
        }

        stage('Publicar en Docker Hub') {
            steps {
                withCredentials([
                    usernamePassword(
                        credentialsId: 'dockerhub',
                        usernameVariable: 'DH_USUARIO',
                        passwordVariable: 'DH_TOKEN'
                    )
                ]) {
                    sh '''#!/bin/bash
                        set -euo pipefail

                        trap 'docker logout >/dev/null 2>&1 || true' EXIT

                        echo "$DH_TOKEN" | docker login \
                            -u "$DH_USUARIO" \
                            --password-stdin

                        docker push "$IMAGEN"
                    '''
                }
            }
        }

        stage('Desplegar en Kubernetes') {
            steps {
                sh '''#!/bin/bash
                    set -euxo pipefail

                    # Crear el namespace si todavía no existe.
                    kubectl create namespace "$NAMESPACE" \
                        --dry-run=client -o yaml | kubectl apply -f -

                    # Motivo del despliegue, visible en "kubectl rollout history".
                    CAUSA="Jenkins build $BUILD_NUMBER - rama $BRANCH_NAME - commit $GIT_CORTO"

                    # Aplicar el Deployment con la imagen, la versión y el motivo actuales.
                    sed -e "s|__IMAGEN__|$IMAGEN|" \
                        -e "s|__VERSION__|$VERSION|" \
                        -e "s|__CAUSA__|$CAUSA|" \
                        k8s/deployment.yaml \
                        | kubectl apply -n "$NAMESPACE" -f -

                    # Aplicar el Service NodePort correspondiente.
                    sed -e "s|__NODEPORT__|$NODE_PORT|" \
                        k8s/service.yaml \
                        | kubectl apply -n "$NAMESPACE" -f -

                    # Esperar a que Kubernetes complete el despliegue.
                    kubectl rollout status \
                        deployment/zerodowntime \
                        -n "$NAMESPACE" \
                        --timeout=180s
                '''
            }
        }

        stage('Verificar despliegue') {
            steps {
                sh '''#!/bin/bash
                    set -euxo pipefail

                    # Mostrar el estado del entorno desplegado.
                    kubectl get deployment,pods,service \
                        -n "$NAMESPACE" -o wide

                    # Obtener la IP interna del nodo Minikube.
                    IP_NODO=$(kubectl get nodes \
                        -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')

                    URL="http://$IP_NODO:$NODE_PORT"
                    echo "Prueba de humo contra $URL"

                    # Comprobar que la aplicación responde correctamente.
                    curl -fsS "$URL/health"
                    echo

                    # Esperar a que todas las respuestas sean de la versión nueva.
                    # Durante unos segundos los pods antiguos siguen atendiendo mientras se retiran.
                    set +x
                    antiguas=1
                    for intento in $(seq 1 15); do
                        antiguas=0
                        for i in 1 2 3 4 5 6; do
                            respuesta=$(curl -fsS "$URL/pod" || true)
                            case "$respuesta" in
                                *"versión: $VERSION"*) ;;
                                *) antiguas=$((antiguas + 1)) ;;
                            esac
                        done
                        if [ "$antiguas" -eq 0 ]; then
                            echo "Intento $intento: todas las respuestas son de la versión $VERSION"
                            break
                        fi
                        echo "Intento $intento: $antiguas de 6 respuestas aún de la versión anterior, reintentando..."
                        sleep 2
                    done
                    if [ "$antiguas" -ne 0 ]; then
                        echo "ERROR: el Service sigue sirviendo una versión distinta de $VERSION"
                        exit 1
                    fi
                    set -x

                    # Mostrar el reparto entre los pods de la versión nueva.
                    for i in 1 2 3 4 5 6; do
                        curl -fsS "$URL/pod"
                    done

                    kubectl rollout history deployment/zerodowntime -n "$NAMESPACE"
                '''
            }
        }
    }

    post {
        success {
            echo "Desplegada la versión ${env.VERSION} en el namespace ${env.NAMESPACE}"
        }

        failure {
            echo "Pipeline fallido. Estado actual del entorno ${env.NAMESPACE ?: 'desconocido'}:"

            sh '''#!/bin/bash
                if [ -n "${NAMESPACE:-}" ]; then
                    kubectl get deployment,pods \
                        -n "$NAMESPACE" -o wide || true

                    kubectl rollout history \
                        deployment/zerodowntime \
                        -n "$NAMESPACE" || true
                fi
            '''
        }

        always {
            echo 'Pipeline finalizado. Las imágenes locales se conservan para facilitar las comprobaciones.'
        }
    }
}
