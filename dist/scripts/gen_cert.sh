#!/bin/bash

# define some variables
CA_DIR="ca"
SERVER_DIR="server"
CLIENT_DIR="client"
DAYS_VALID=365
COUNTRY="CN"
STATE="Zhejiang"
LOCALITY="Hangzhou"
ORGANIZATION="singchia"
ORGANIZATIONAL_UNIT="frontier"
COMMON_NAME_CA="MyCA"
COMMON_NAME_SERVER="server.frontier.com"
COMMON_NAME_CLIENT="client.frontier.com"

# make directories
mkdir -p ${CA_DIR} ${SERVER_DIR} ${CLIENT_DIR}

# gen ca cert and key
openssl genpkey -algorithm RSA -out ${CA_DIR}/ca.key
openssl req -x509 -new -nodes -key ${CA_DIR}/ca.key -sha256 -days ${DAYS_VALID} -out ${CA_DIR}/ca.crt -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORGANIZATION}/OU=${ORGANIZATIONAL_UNIT}/CN=${COMMON_NAME_CA}"

# gen server key and csr
openssl genpkey -algorithm RSA -out ${SERVER_DIR}/server.key
openssl req -new -key ${SERVER_DIR}/server.key -out ${SERVER_DIR}/server.csr -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORGANIZATION}/OU=${ORGANIZATIONAL_UNIT}/CN=${COMMON_NAME_SERVER}"

# gen server cert
openssl x509 -req -in ${SERVER_DIR}/server.csr -CA ${CA_DIR}/ca.crt -CAkey ${CA_DIR}/ca.key -CAcreateserial -out ${SERVER_DIR}/server.crt -days ${DAYS_VALID} -sha256

# gen client key and csr
openssl genpkey -algorithm RSA -out ${CLIENT_DIR}/client.key
openssl req -new -key ${CLIENT_DIR}/client.key -out ${CLIENT_DIR}/client.csr -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORGANIZATION}/OU=${ORGANIZATIONAL_UNIT}/CN=${COMMON_NAME_CLIENT}"

# gen client cert
openssl x509 -req -in ${CLIENT_DIR}/client.csr -CA ${CA_DIR}/ca.crt -CAkey ${CA_DIR}/ca.key -CAcreateserial -out ${CLIENT_DIR}/client.crt -days ${DAYS_VALID} -sha256

echo "CA, Server, and Client certificates and keys have been generated."
