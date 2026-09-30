# Circuito-API

API HTTP de eventos de corrida, em Go. Substitui a API FastAPI anterior
mantendo o mesmo contrato; scrapers e Selenium continuam em Python,
em imagem e execução separadas.

## Arquitetura

```text
Frontend
   |
   v
API HTTP em Go (este repo)
   |
   +--> MongoDB (eventos, jobs, sync)
   +--> S3/bucket (eventos_real.json)
   +--> dispara/consulta worker de scraping (Python, separado)
```

## Endpoints

Públicos: `GET /health`, `GET /ready`, `GET /api/v1/eventos`,
`GET /api/v1/eventos/{id}`, `GET /api/v1/dashboard/stats`.

Protegidos por `X-API-Key`: `POST/PATCH/DELETE /api/v1/eventos`,
`GET/POST /api/v1/sync-bucket(/status)`.

Protegidos por `X-API-Key` de scrapers: `POST /api/v1/scrape/run`,
`GET /api/v1/scrape/status/{id}`, `GET /api/v1/scrape/last-run`,
`POST /api/v1/scrape/import`.

Contrato completo em [`docs/api-contract.md`](docs/api-contract.md).

## Configuração

Toda configuração chega por variáveis de ambiente. O arquivo `.env`
é opcional e usado apenas em desenvolvimento local (nunca commitado,
nunca copiado para a imagem Docker).

| Variável | Default | Descrição |
|---|---|---|
| `MONGODB_URI` | `mongodb://localhost:27017` | Conexão MongoDB/Atlas |
| `MONGODB_DB_NAME` | `corridas_db` | Banco |
| `MONGODB_COLLECTION` | `eventos` | Collection de eventos |
| `API_PORT` | `8181` | Porta HTTP |
| `API_KEY` | — | Chave das rotas de escrita |
| `SCRAPERS_API_KEY` | — | Chave das rotas de scrape |
| `CORS_ORIGINS` | `*` | Origens permitidas (CSV) |
| `AWS_BUCKET_NAME` | — | Bucket do sync |
| `AWS_REGION` | `us-east-1` | Região AWS |
| `AWS_ACCESS_KEY_ID` | — | Credencial AWS |
| `AWS_SECRET_ACCESS_KEY` | — | Credencial AWS |
| `BUCKET_JSON_KEY` | `eventos_real.json` | Objeto do sync |

## Desenvolvimento local

```bash
go run ./cmd/api
```

## Verificações

```bash
gofmt -l .
go vet ./...
go test ./... -count=1
go test -race ./... -count=1
go build ./...
```

## Docker

```bash
docker build -t circuito-api:local .
docker run --rm -p 8181:8181 \
  -e API_PORT=8181 \
  circuito-api:local
```

A imagem final é `distroless/static-debian12:nonroot`: só o binário,
sem shell, sem Chromium, sem Python. Ela roda como usuário não-root
e não contém nenhum segredo — toda configuração chega por ambiente.
