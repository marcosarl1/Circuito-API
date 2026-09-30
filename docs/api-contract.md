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

As rotas de escrita exigem o header:

```http
X-API-Key: <chave-secreta>
```

As rotas de leitura e health não exigem autenticação.

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
