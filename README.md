curl -i -X POST http://localhost:8080/api/catalina \
    --output files.zip \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-01-26T11:07:00+03:00"}'


curl -i -X POST http://localhost:8080/api/scaners \
    --output files.zip \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-01-27T11:07:00+03:00","scanid":"test"}'


