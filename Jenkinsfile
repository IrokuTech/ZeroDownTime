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

                    # Aplicar el Deployment con la imagen y versión actuales.
                    sed -e "s|__IMAGEN__|$IMAGEN|" \
                        -e "s|__VERSION__|$VERSION|" \
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

                    echo "Prueba de humo contra http://$IP_NODO:$NODE_PORT"

                    # Comprobar que la aplicación responde correctamente.
                    curl -fsS "http://$IP_NODO:$NODE_PORT/health"
                    echo

                    # Comprobar la identificación de los pods.
                    for i in 1 2 3 4 5 6; do
                        curl -fsS "http://$IP_NODO:$NODE_PORT/pod"
                    done
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
