# Contrato da API de Eventos

Este documento descreve o contrato atual da API de eventos.
Ele deve ser atualizado sempre que uma resposta pública mudar.

## Base URL

As rotas de eventos começam com:

```text
/api/v1
```

## Identificador

O campo público oficial é:

```json
{
  "id": "2026090001"
}
```

Regras:

- `id` é uma string;
- novos IDs usam o formato `YYYYMM####`;
- o valor do `id` deve ser igual ao `_id` persistido no MongoDB;
- não usar o índice da lista como identificador;
- não converter o ID para número;
- não remover zeros à esquerda;
- o mesmo `id` deve funcionar em `GET`, `PATCH` e `DELETE`.

## Autenticação

Rotas de leitura e health não exigem autenticação.

### Usuários (JWT)

Fluxo para o painel admin, com usuário e senha como única credencial:

1. `POST /api/v1/auth/login` `{"username","password"}` → `200`
   `{access_token, refresh_token, expires_in, username}` + cookie
   `refresh_token` (HttpOnly, regra 5/min por cliente).
2. Requisições autenticadas enviam `Authorization: Bearer <access_token>`
   (15 min de validade).
3. `POST /api/v1/auth/refresh` (cookie ou `{"refresh_token"}`) → novo par;
   o token apresentado é consumido (reuso sempre 401).
4. `POST /api/v1/auth/logout` → `204`, revoga e limpa o cookie.
5. `GET /api/v1/auth/me` → conta no formato público.
6. `PATCH /api/v1/auth/password` → troca com senha atual (mínimo 8).

Erros: credencial errada `401 {"detail":"Credenciais inválidas"}`
(mesma mensagem para usuário inexistente); conta desabilitada `403`;
refresh inválido/usado `401 {"detail":"Sessão inválida"}`.

Contas vivem na collection `users` (`username` único, `password_hash`
bcrypt, `role`). O `_id` é ObjectID nativo (hex de 24 chars no JSON).
Não há seed automático: a conta admin é criada manualmente uma vez
(ver README). Rotas de escrita exigem JWT de conta `ADMIN` (qualquer outra
role recebe `403`); rotas self-service (`/me`, troca de senha) aceitam
qualquer conta válida.

### Chaves de serviço (X-API-Key)

Rotas de escrita aceitam JWT **ou** `X-API-Key` durante a transição do
frontend (remover o caminho da chave depois). Rotas de scrape exigem
somente a chave de scrapers (integrações máquina-máquina).

## Endpoints

### GET /health

Verifica se o processo da API está vivo.

Resposta:

```json
{
  "status": "ok"
}
```

### GET /ready

Verifica se a API está pronta para receber tráfego.

Quando o MongoDB está disponível:

```json
{
  "status": "ready"
}
```

Quando o MongoDB não está disponível:

```json
{
  "status": "not_ready",
  "reason": "db not connected"
}
```

### GET /api/v1/eventos

Lista eventos.

Parâmetros:

- `page`: número da página, começando em `1`;
- `size`: quantidade de itens por página, com padrão `20` e máximo `100`;
- `estado`: filtro por UF, quando informado;
- `q`: busca textual, quando informada.

Resposta:

```json
{
  "eventos": [],
  "total": 0,
  "total_pages": 0,
  "page": 1,
  "size": 20,
  "has_next": false,
  "has_prev": false
}
```

Não existe limite artificial para a quantidade de páginas.
Se for solicitada uma página depois do último resultado, `eventos` será uma lista vazia.

A ordenação atual é por `datas_realizacao` em ordem decrescente.

### GET /api/v1/eventos/{id}

Busca um evento pelo campo público `id`.

Exemplo:

```http
GET /api/v1/eventos/2026090001
```

Resposta de sucesso:

```json
{
  "id": "2026090001",
  "nome_evento": "Corrida de Teste",
  "cidade": "João Pessoa",
  "estado": "PB"
}
```

Quando o evento não existe:

```http
404 Not Found
```

```json
{
  "detail": "Evento não encontrado"
}
```

### POST /api/v1/eventos

Cria um evento.

Requer autenticação.

Exemplo de requisição:

```json
{
  "nome_evento": "Corrida de Teste",
  "cidade": "João Pessoa",
  "estado": "PB",
  "datas_realizacao": [
    "2026-09-24T00:00:00Z"
  ]
}
```

Resposta de sucesso:

```http
201 Created
```

A resposta deve conter o evento criado e o `id` gerado.

### PATCH /api/v1/eventos/{id}

Atualiza parcialmente um evento.

Requer autenticação.

Exemplo de requisição:

```json
{
  "cidade": "Campina Grande"
}
```

Resposta de sucesso:

```http
200 OK
```

A resposta deve conter o evento atualizado.

Se nenhum campo for enviado:

```http
400 Bad Request
```

```json
{
  "detail": "Nenhum campo para atualizar"
}
```

### DELETE /api/v1/eventos/{id}

Remove um evento.

Requer autenticação.

Se o evento existir:

```http
204 No Content
```

Depois da remoção, o mesmo ID deve retornar:

```http
404 Not Found
```

### POST /api/v1/scrape/run

Solicita uma execução dos scrapers. Requer `X-API-Key` com a chave de scrapers.

Resposta:

```http
202 Accepted
```

```json
{
  "job_id": "e840f07fb7db5b74c18cbfae95942f4d"
}
```

Se já existir um job em andamento (`queued` ou `running`):

```http
409 Conflict
```

```json
{
  "detail": "Scrape já está em andamento"
}
```

Quando o disparo automático está ligado (`SCRAPER_TRIGGER_ENABLED=true`),
o `POST` inicia a execução no Azure sozinho: o frontend continua só
aguardando o polling, sem passo manual. Se o disparo falhar, o job é
abandonado (`failed`, slot liberado) e a resposta é:

```http
503 Service Unavailable
```

```json
{
  "detail": "Serviço de scraping indisponível"
}
```

Pré-requisitos do disparo (executados uma vez, fora da API): identidade
gerenciada no app da API + role `Container Apps Jobs Operator` no job.
Sem isso (desenvolvimento local), o comportamento é só-enfileirar.

### Fluxo com confirmação humana

A importação exige confirmação antes de gravar no banco. O ciclo é:

1. `POST /scrape/run` → `202 {job_id}` (coleta executa).
2. Job vai para `awaiting_import` com o relatório parcial; o frontend
   abre o modal de revisão (qualquer estado não terminal mantém polling).
3. `POST /api/v1/scrape/confirm/{id}` → `202` (requer chave de scrapers):
   o job volta a `queued` para a fase de import e o worker dispara.
   Estado errado → `409`; inexistente → `404`.
4. `POST /api/v1/scrape/cancel/{id}` → `204`: abandona o job, **apaga os
   CSVs do payload** e libera o slot (dados descartados não esperam TTL).
   Inexistente → `404`.

### Recuperar coleta pendente

Reload ou modal fechado durante a revisão não perdem a coleta: o payload
fica no Mongo por TTL de 24h.

```http
GET /api/v1/scrape/awaiting
```

```json
{
  "job": {
    "job_id": "e840f07fb7db5b74c18cbfae95942f4d",
    "status": "awaiting_import",
    "phase": "collect",
    "started_at": "2026-10-08T02:11:16.000000+00:00",
    "finished_at": "",
    "report": { "scrapers": [], "csvs": [] },
    "error": null
  }
}
```

`job: null` quando não há nada pendente. Com job presente, o painel reabre
o modal e oferece confirmar ou descartar — mesma semântica do passo 3/4.

Job confirmado tem o payload apagado após a importação concluir; job
descartado tem o payload apagado no ato. Só uma run sem decisão (sessão
fechada) mantém o payload até o TTL.

Sem confirmação ou cancelamento, o slot permanece ocupado: duas
requisições simultâneas nunca iniciam duas execuções.

A aquisição é atômica: duas réplicas não iniciam duas execuções. Limite de 5 chamadas por minuto por cliente (`429`).

O job fica persistido no MongoDB (`scrape_jobs`) com status `queued` até o worker Python existir e conduzi-lo para `running` → `complete`/`failed`.

### GET /api/v1/scrape/status/{id}

Retorna o job persistido. Requer chave de scrapers.

```json
{
  "job_id": "e840f07fb7db5b74c18cbfae95942f4d",
  "status": "queued",
  "started_at": "2026-09-30T02:13:55.849597+00:00",
  "finished_at": "",
  "report": null,
  "error": null
}
```

`status` usa os mesmos valores da API anterior (`running`, `complete`, `failed`), acrescido de `queued` para jobs aguardando o worker. O frontend trata qualquer estado não terminal como em andamento.

Job inexistente:

```http
404 Not Found
```

### Formato do `report` de scrape

O `GET /api/v1/scrape/status/{id}` devolve o relatório no formato
`ScrapeReport` consumido pelo painel admin (`scrapers[]` + `csvs[]`):

```json
{
  "job_id": "e840f07fb7db5b74c18cbfae95942f4d",
  "status": "complete",
  "started_at": "2026-10-08T02:00:03.713390+00:00",
  "finished_at": "2026-10-08T02:11:51.086188+00:00",
  "report": {
    "started_at": "2026-10-08T02:11:16.000000+00:00",
    "finished_at": "2026-10-08T02:11:41.000000+00:00",
    "scrapers": [
      {
        "nome": "scraper_brasilquecorre.py",
        "ok": true,
        "duration_s": 4.5,
        "detail": "...",
        "stderr": ""
      }
    ],
    "csvs": [
      {
        "fonte": "brasilquecorre",
        "ok": true,
        "total": 2,
        "duplicados": 0,
        "sem_preco": 0,
        "eventos_passados": 0,
        "sem_imagem": 0,
        "erros_encoding": 0,
        "erros": []
      }
    ]
  },
  "error": null
}
```

Regras: `detail` limitado a 8000 e `stderr` a 2000 caracteres, `erros`
limitado a 10 itens (mesmos truncamentos do fluxo anterior). Quando o
relatório estruturado não existe (jobs antigos), `report` contém
`scrapers: []` e `csvs: []` e o modal exibe as seções vazias.

### GET /api/v1/scrape/last-run

Requer chave de scrapers.

```json
{
  "finished_at": null
}
```

Retorna `null` quando nenhuma coleta foi concluída ainda.

### POST /api/v1/scrape/import

Requer chave de scrapers. A importação depende do worker Python, que ainda não existe:

```http
501 Not Implemented
```

```json
{
  "detail": "Importação indisponível: worker de scraping não configurado"
}
```

### GET /api/v1/sync-bucket/status

Informa se uma sincronização com o bucket está em andamento. Requer `X-API-Key`.

```json
{
  "in_progress": false
}
```

### POST /api/v1/sync-bucket

Serializa todos os eventos para o formato legado do bucket (chave `_id`, igual ao arquivo `eventos_real.json` consumido pelo site público) e envia ao S3. Requer `X-API-Key`.

Sucesso:

```http
200 OK
```

```json
{
  "status": "ok",
  "eventos_synced": 246
}
```

Quando o conteúdo é idêntico ao último sync, o upload é pulado:

```json
{
  "status": "unchanged",
  "eventos_synced": 246
}
```

O `unchanged` é uma extensão compatível: mesmo formato, um valor novo. O fingerprint (SHA256) fica persistido no MongoDB (`bucket_sync`), então sobrevive a reinícios.

Sem bucket configurado:

```http
500 Internal Server Error
```

```json
{
  "detail": "AWS_BUCKET_NAME não configurado"
}
```

Sync em andamento:

```http
409 Conflict
```

### GET /api/v1/dashboard/stats

Retorna estatísticas agregadas dos eventos. Rota pública, sem autenticação.

Resposta (chaves principais):

```json
{
  "total": 246,
  "ativos": 59,
  "passados": 187,
  "proximos30d": 12,
  "proximos90d": 30,
  "semPreco": 40,
  "patrocinados": 5,
  "semImagem": 10,
  "semLink": 3,
  "semRegulamento": 20,
  "valorMedio": 89.72,
  "lote1Count": 15,
  "porMes": [{"label": "2026-10", "count": 8}],
  "porEstado": [{"estado": "PB", "count": 228}],
  "porCidade": [{"cidade": "João Pessoa", "count": 150}],
  "porDistancia": [{"distancia": "5KM", "count": 120}],
  "porOrganizador": [{"organizador": "Org A", "count": 30}],
  "porFonte": [{"fonte": "brasilquecorre", "count": 200}],
  "densidade": [{"data": "2026-10-04", "count": 2}],
  "choques": 4,
  "statusInscricoes": {"abertas": 40, "emBreve": 19, "encerradas": 187},
  "comPercurso": 50,
  "comKits": 30,
  "porHorario": [{"label": "07:00", "count": 90}],
  "porKit": [{"label": "Kit Básico", "count": 20}],
  "scraperHealth": [],
  "proximosEventos": []
}
```

Regras:

- `proximosEventos` usa `_id` (com underline) como chave do identificador, igual ao backend anterior;
- `statusInscricoes.emBreve` usa camelCase;
- `scraperHealth` exclui fontes manuais e `ticketsports`.

## Formato de erro

Os erros usam o formato:

```json
{
  "detail": "Mensagem de erro"
}
```

A API não deve retornar:

- stack trace;
- URI do MongoDB;
- usuário ou senha;
- mensagem interna do driver MongoDB;
- credenciais de ambiente.

## Campos de evento

Exemplo completo de referência:

```json
{
  "id": "2026090001",
  "nome_evento": "Corrida de Teste",
  "cidade": "João Pessoa",
  "estado": "PB",
  "organizador": "Organização de Teste",
  "data_realizacao": "24 de Setembro de 2026",
  "datas_realizacao": [
    "2026-09-24T00:00:00Z"
  ],
  "distancias": [
    "5 KM",
    "10 KM"
  ],
  "horario": "07:00",
  "url_inscricao": "https://example.com/inscricao",
  "url_imagem": "https://example.com/imagem.jpg",
  "link_edital": "https://example.com/edital.pdf",
  "precos_entries": [
    "Lote 1 - R$ 120,00"
  ],
  "patrocinado": false,
  "percurso": {
    "local_largada": "Praça Central",
    "trajeto": "Avenida Principal"
  },
  "kits": [],
  "campos_protegidos": []
}
```

## Compatibilidade legada

Eventos antigos podem ter menos campos ou formatos diferentes.
A API deve tentar localizá-los sem alterar seus dados.
Novos eventos devem seguir somente o formato canônico `YYYYMM####`.
