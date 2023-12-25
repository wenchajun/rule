#!/bin/bash
curl -XPOST -H "Content-Type: application/json" -d @auditing.json http://127.0.0.1:8080/webhook/auditing