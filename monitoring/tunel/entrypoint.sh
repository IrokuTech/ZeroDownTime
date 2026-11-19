#!/bin/sh
# Tunel SSH persistente hacia la EC2.
#  - 9100  -> node_exporter (127.0.0.1:9100 en la EC2)
#  - 30080 -> app en namespace produccion (NodePort en Minikube)
#  - 30081 -> app en namespace pruebas    (NodePort en Minikube)
# ExitOnForwardFailure y ServerAlive* hacen que ssh termine si el tunel deja de
# funcionar; Docker lo reinicia (restart: unless-stopped).
set -e

# La clave montada desde WSL puede tener permisos que ssh rechaza: se copia con 0600.
cp /ssh/ec2.pem /tmp/ec2.pem
chmod 600 /tmp/ec2.pem

echo "Abriendo tunel SSH hacia ${EC2_USER}@${EC2_HOST}"
exec ssh -N \
  -i /tmp/ec2.pem \
  -o UserKnownHostsFile=/ssh/known_hosts \
  -o StrictHostKeyChecking=yes \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=15 \
  -o ServerAliveCountMax=3 \
  -L 0.0.0.0:9100:localhost:9100 \
  -L 0.0.0.0:30080:${MINIKUBE_IP}:30080 \
  -L 0.0.0.0:30081:${MINIKUBE_IP}:30081 \
  "${EC2_USER}@${EC2_HOST}"
