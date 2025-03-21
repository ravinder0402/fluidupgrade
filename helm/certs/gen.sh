#!/bin/bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 3650 -nodes -addext "subjectAltName=DNS:*.compass-system,IP:127.0.0.1"
