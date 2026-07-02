# downlog

Простой HTTP-сервис для скачивания архивов логов по дате или ScanID.

## Что умеет

- Возвращает ZIP-архив с логами для endpoints:
  - `/api/catalina`
  - `/api/universe`
  - `/api/scaners`
  - `/api/scaners-formit`
  - `/api/scaners-logtxt`
- При отсутствии подходящих файлов возвращает JSON-ошибку с кодом `404` и сообщением:
  - `{"error":"Файл не найден"}`


Сервис ожидает TLS-сертификаты по путям из переменных окружения `DL_CERT` и `DL_KEY`.

## Примеры запросов

### HTTP

```bash
curl -i -X POST http://localhost:8080/api/catalina \
  --output files.zip \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-01-26T11:07:00+03:00"}'
```

```bash
curl -i -X POST http://localhost:8080/api/scaners \
  --output files.zip \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-01-27T11:07:00+03:00","scanid":"test"}'
```

### HTTPS с сертификатом

```bash
curl -i -X POST https://localhost:8080/api/catalina \
  --output files.zip \
  --cert cert.crt \
  --key privet.key \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-01-26T11:07:00+03:00"}'
```


## Формат запроса

Для `catalina` и `universe` ожидается JSON с полем `timestamp` в формате RFC3339:

```json
{"timestamp":"2026-01-26T11:07:00+03:00"}
```

Для `scaners` и `scaners-formit` ожидается JSON с полем `scanid`:

```json
{"scanid":"test"}
```

## Переменные окружения backend

Сервис читает следующие переменные окружения:

- `UPT_LIMIT_DOWNLOAD_MB` — лимит размера загрузки в мегабайтах, по умолчанию `500`
- `DL_URL_API_PREFIX` — префикс URL API, по умолчанию пустая строка
- `DL_SCAN_LOG` — путь к логам scaners, по умолчанию `/app/edm/scan/logs`
- `DL_SCAN_LOG_FORMIT` — путь к логам scaners-formit, по умолчанию `/app/edm/scan/logs`
- `DL_SCAN_LOG_LOGTXT` — путь к логам log.txt, по умолчанию `/app/edm/scan/logs`
- `DL_TOMCAT` — путь к логам Tomcat, по умолчанию `/app/edm/tomcat-9/logs`
- `DL_LISTEN_ADDR` — адрес и порт слушателя, по умолчанию `localhost:8080`
- `DL_CERT` — путь к TLS-сертификату, по умолчанию `/certs/cert.crt`
- `DL_KEY` — путь к TLS-ключу, по умолчанию `/certs/privet.key`
