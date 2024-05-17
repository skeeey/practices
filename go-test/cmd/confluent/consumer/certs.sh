#!/bin/bash


kubectl -n open-cluster-management-agent get secrets maestro-addon-open-cluster-management.io-maestro-addon-client-cert -ojsonpath='{.data.tls\.crt}' | base64 -d > client.crt
kubectl -n open-cluster-management-agent get secrets maestro-addon-open-cluster-management.io-maestro-addon-client-cert -ojsonpath='{.data.tls\.key}' | base64 -d > client.key
kubectl -n open-cluster-management-agent get secrets maestro-mq-ca -ojsonpath='{.data.ca\.crt}' | base64 -d > ca.crt