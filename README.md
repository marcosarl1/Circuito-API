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
| `API_KEY` | — | Chave das rotas de escrita (transição; serviços) |
| `SCRAPERS_API_KEY` | — | Chave das rotas de scrape |
| `JWT_SECRET` | — | Segredo HMAC dos tokens (32+ bytes aleatórios) |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | — | Seed do primeiro admin |
| `CORS_ORIGINS` | `*` | Origens permitidas (CSV) |
| `AWS_BUCKET_NAME` | — | Bucket do sync |
| `AWS_REGION` | `us-east-1` | Região AWS |
| `AWS_ACCESS_KEY_ID` | — | Credencial AWS |
| `AWS_SECRET_ACCESS_KEY` | — | Credencial AWS |
| `BUCKET_JSON_KEY` | `eventos_real.json` | Objeto do sync |
| `SCRAPER_TRIGGER_ENABLED` | (vazio) | `true` liga o disparo automático do worker |
| `AZURE_SUBSCRIPTION_ID` | — | Subscription do job (só com trigger ligado) |
| `AZURE_RESOURCE_GROUP` | `rg-circuitoapp` | Resource group do job |
| `SCRAPER_JOB_NAME` | `correpb-scraper` | Nome do Container Apps Job |

## Desenvolvimento local

```bash
go run ./cmd/api
```

## Verificações

```bash
gofmt -l .
go vet ./...
go test ./... -count=1          # unitários (rápidos, sem Docker)
go test -race ./... -count=1
```

Integração e performance sobem a stack real sozinhas (Mongo em
container + binário da API, sem mocks, sem servidor remoto).
Requer apenas Docker em execução:

```bash
go test -tags integration ./tests/integration/ -v -count=1
go test -tags perf ./tests/performance/ -v -count=1
```

O CI executa as três suítes; o deploy valida a revisão nova com
smoke test (`/health`, `/ready` e forma da listagem).

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

## Deploy

Push para `main` com CI verde publica a imagem no GHCR (pacote
**privado**, tag pelo SHA do commit) e atualiza o Container App
usando o **digest** da imagem, nunca tag mutável.

O app e o resource group vêm das variables `AZURE_CONTAINERAPP_NAME`
(default `circuito-api`) e `AZURE_RESOURCE_GROUP` (default
`rg-circuitoapp`). Crie o app antes do primeiro deploy (staging);
o tráfego só migra para a API Go na fase de cutover.

Secrets necessários no repositório (nomes canônicos, sem as
variações legadas `MONGO_DB_NAME`/`MONGODB_REMOTE_DB`):

`AZURE_CREDENTIALS`, `MONGODB_URI`, `MONGODB_DB_NAME`,
`MONGODB_COLLECTION`, `API_KEY`, `SCRAPERS_API_KEY`, `CORS_ORIGINS`,
`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
`AWS_BUCKET_NAME`, `GHCR_USERNAME`, `GHCR_PAT` (PAT com
`read:packages`, para o pull da imagem privada).

Rollback: como as tags são imutáveis por SHA, volte o app para
o digest anterior:

```bash
az containerapp update \
  -n circuito-api -g rg-circuitoapp \
  -i "ghcr.io/marcosarl1/circuito-api@sha256:<digest-anterior>"
```

O digest de cada deploy fica no log do workflow.
