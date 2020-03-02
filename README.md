# load test http

> echo "GET http://localhost:8081/" | vegeta attack -duration=60s -rate=250/s | vegeta encode > results.json

> vegeta report results.*