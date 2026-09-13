-- Schema tse_* (espelho da estrutura usada no projeto odp, sem convenios).
CREATE TABLE tse_eleicao (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo_tse          INTEGER NOT NULL,
    ano                 SMALLINT,
    codigo_tipo_eleicao INTEGER,
    nome_tipo_eleicao   TEXT,
    descricao           TEXT,
    data_eleicao        DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT tse_eleicao_codigo_tse_uq UNIQUE (codigo_tse)
);

CREATE TABLE tse_unidade_eleitoral (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sg_uf      TEXT NOT NULL,
    codigo_tse TEXT NOT NULL,
    nome       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT tse_unidade_uq UNIQUE (sg_uf, codigo_tse)
);

CREATE TABLE tse_partido (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero                  SMALLINT NOT NULL,
    sigla                   TEXT,
    nome                    TEXT,
    federacao_codigo_tse    BIGINT,
    federacao_sigla         TEXT,
    federacao_nome          TEXT,
    coligacao_codigo_tse    BIGINT,
    coligacao_nome          TEXT,
    coligacao_composicao    TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT tse_partido_numero_uq UNIQUE (numero)
);

CREATE TABLE tse_candidato (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sq_candidato                    BIGINT NOT NULL,
    eleicao_id                      UUID NOT NULL REFERENCES tse_eleicao(id),
    sg_uf                           TEXT NOT NULL,
    partido_id                      UUID REFERENCES tse_partido(id),
    cargo_codigo                    INTEGER,
    cargo_nome                      TEXT,
    genero_descricao                TEXT,
    cor_raca_descricao              TEXT,
    estado_civil_nome               TEXT,
    grau_instrucao_nome             TEXT,
    ocupacao_codigo                 INTEGER,
    ocupacao_nome                   TEXT,
    numero_candidato                INTEGER,
    cpf                             TEXT,
    cpf_vice                        TEXT,
    nome_completo                   TEXT NOT NULL,
    nome_urna                       TEXT,
    nome_social                     TEXT,
    data_nascimento                 DATE,
    situacao_totalizacao_descricao  TEXT,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                      TIMESTAMPTZ,
    CONSTRAINT tse_candidato_sq_uq UNIQUE (sq_candidato)
);

CREATE TABLE tse_fornecedor (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cpf_cnpj                        TEXT NOT NULL,
    nome                            TEXT,
    nome_rfb                        TEXT,
    tipo_fornecedor_codigo          INTEGER,
    tipo_fornecedor_descricao       TEXT,
    cnae_codigo                     TEXT,
    cnae_descricao                  TEXT,
    esfera_partidaria_codigo        TEXT,
    esfera_partidaria_descricao     TEXT,
    sg_uf                           TEXT,
    municipio_nome                  TEXT,
    sq_candidato_relacionado        BIGINT,
    numero_candidato_relacionado    INTEGER,
    cargo_codigo_relacionado        INTEGER,
    cargo_descricao_relacionada     TEXT,
    partido_numero_relacionado      SMALLINT,
    partido_sigla_relacionado       TEXT,
    partido_nome_relacionado        TEXT,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                      TIMESTAMPTZ,
    CONSTRAINT tse_fornecedor_cnpj_uq UNIQUE (cpf_cnpj)
);

CREATE TABLE tse_doador (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cpf_cnpj                        TEXT NOT NULL,
    nome                            TEXT,
    nome_rfb                        TEXT,
    cnae_codigo                     TEXT,
    cnae_descricao                  TEXT,
    esfera_partidaria_codigo        TEXT,
    esfera_partidaria_descricao     TEXT,
    sg_uf                           TEXT,
    municipio_nome                  TEXT,
    sq_candidato_relacionado        BIGINT,
    numero_candidato_relacionado    INTEGER,
    cargo_codigo_relacionado        INTEGER,
    cargo_descricao_relacionada     TEXT,
    partido_numero_relacionado      SMALLINT,
    partido_sigla_relacionado       TEXT,
    partido_nome_relacionado        TEXT,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                      TIMESTAMPTZ,
    CONSTRAINT tse_doador_cnpj_uq UNIQUE (cpf_cnpj)
);

CREATE TABLE tse_prestacao_contas (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sq_prestador_contas         BIGINT NOT NULL,
    eleicao_id                  UUID NOT NULL REFERENCES tse_eleicao(id),
    candidato_id                UUID REFERENCES tse_candidato(id),
    partido_id                  UUID REFERENCES tse_partido(id),
    sg_uf                       TEXT,
    unidade_eleitoral_id        UUID REFERENCES tse_unidade_eleitoral(id),
    tipo_prestador              TEXT NOT NULL,
    tipo_prestacao              TEXT,
    data_prestacao              DATE,
    turno                       SMALLINT,
    cnpj_prestador_conta        TEXT,
    esfera_partidaria_codigo    TEXT,
    esfera_partidaria_descricao TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ,
    CONSTRAINT tse_prestacao_uq UNIQUE (tipo_prestador, eleicao_id, sq_prestador_contas)
);

CREATE TABLE tse_despesa_candidato (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id         UUID NOT NULL REFERENCES tse_prestacao_contas(id),
    candidato_id                UUID NOT NULL REFERENCES tse_candidato(id),
    fornecedor_id               UUID REFERENCES tse_fornecedor(id),
    sq_despesa                  BIGINT NOT NULL,
    tipo_registro               TEXT NOT NULL,
    tipo_documento              TEXT,
    numero_documento            TEXT,
    origem_despesa_codigo       INTEGER,
    origem_despesa_descricao    TEXT,
    fonte_despesa_codigo        INTEGER,
    fonte_despesa_descricao     TEXT,
    natureza_despesa_codigo     INTEGER,
    natureza_despesa_descricao  TEXT,
    especie_recurso_codigo      INTEGER,
    especie_recurso_descricao   TEXT,
    sq_parcelamento_despesa     BIGINT,
    data_despesa                DATE,
    descricao                   TEXT,
    valor                       NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ
);

CREATE TABLE tse_despesa_orgao_partidario (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id         UUID NOT NULL REFERENCES tse_prestacao_contas(id),
    partido_id                  UUID NOT NULL REFERENCES tse_partido(id),
    fornecedor_id               UUID REFERENCES tse_fornecedor(id),
    sq_despesa                  BIGINT NOT NULL,
    tipo_registro               TEXT NOT NULL,
    tipo_documento              TEXT,
    numero_documento            TEXT,
    origem_despesa_codigo       INTEGER,
    origem_despesa_descricao    TEXT,
    fonte_despesa_codigo        INTEGER,
    fonte_despesa_descricao     TEXT,
    natureza_despesa_codigo     INTEGER,
    natureza_despesa_descricao  TEXT,
    especie_recurso_codigo      INTEGER,
    especie_recurso_descricao   TEXT,
    sq_parcelamento_despesa     BIGINT,
    data_despesa                DATE,
    descricao                   TEXT,
    valor                       NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ
);

CREATE TABLE tse_receita_candidato (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id         UUID NOT NULL REFERENCES tse_prestacao_contas(id),
    candidato_id                UUID NOT NULL REFERENCES tse_candidato(id),
    doador_id                   UUID REFERENCES tse_doador(id),
    sq_receita                  BIGINT NOT NULL,
    fonte_receita_codigo        INTEGER,
    fonte_receita_descricao     TEXT,
    origem_receita_codigo       INTEGER,
    origem_receita_descricao    TEXT,
    natureza_receita_codigo     INTEGER,
    natureza_receita_descricao  TEXT,
    especie_receita_codigo      INTEGER,
    especie_receita_descricao   TEXT,
    numero_recibo_doacao        TEXT,
    numero_documento_doacao     TEXT,
    data_receita                DATE,
    descricao                   TEXT,
    valor                       NUMERIC(18,2) NOT NULL DEFAULT 0,
    natureza_recurso_estimavel  TEXT,
    genero                      TEXT,
    cor_raca                    TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ
);

CREATE TABLE tse_receita_orgao_partidario (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id         UUID NOT NULL REFERENCES tse_prestacao_contas(id),
    partido_id                  UUID NOT NULL REFERENCES tse_partido(id),
    doador_id                   UUID REFERENCES tse_doador(id),
    sq_receita                  BIGINT NOT NULL,
    fonte_receita_codigo        INTEGER,
    fonte_receita_descricao     TEXT,
    origem_receita_codigo       INTEGER,
    origem_receita_descricao    TEXT,
    natureza_receita_codigo     INTEGER,
    natureza_receita_descricao  TEXT,
    especie_receita_codigo      INTEGER,
    especie_receita_descricao   TEXT,
    numero_recibo_doacao        TEXT,
    numero_documento_doacao     TEXT,
    data_receita                DATE,
    descricao                   TEXT,
    valor                       NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ
);

CREATE TABLE tse_receita_doador_originario_candidato (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id     UUID REFERENCES tse_prestacao_contas(id),
    receita_candidato_id    UUID REFERENCES tse_receita_candidato(id),
    sq_receita              BIGINT NOT NULL,
    documento_doador        TEXT,
    nome_doador             TEXT,
    nome_doador_rfb         TEXT,
    tipo_doador             TEXT,
    cnae_codigo             TEXT,
    cnae_descricao          TEXT,
    data_receita            DATE,
    descricao               TEXT,
    valor                   NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE TABLE tse_receita_doador_originario_orgao_partidario (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestacao_contas_id             UUID REFERENCES tse_prestacao_contas(id),
    receita_orgao_partidario_id     UUID REFERENCES tse_receita_orgao_partidario(id),
    sq_receita                      BIGINT NOT NULL,
    documento_doador                TEXT,
    nome_doador                     TEXT,
    nome_doador_rfb                 TEXT,
    tipo_doador                     TEXT,
    cnae_codigo                     TEXT,
    cnae_descricao                  TEXT,
    data_receita                    DATE,
    descricao                       TEXT,
    valor                           NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                      TIMESTAMPTZ
);

CREATE TABLE tse_bem_candidato (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidato_id            UUID NOT NULL REFERENCES tse_candidato(id),
    tipo_bem_codigo         INTEGER,
    tipo_bem_nome           TEXT,
    numero_ordem            INTEGER NOT NULL,
    descricao               TEXT,
    valor                   NUMERIC(18,2) NOT NULL DEFAULT 0,
    data_ultima_atualizacao DATE,
    hora_ultima_atualizacao TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT tse_bem_uq UNIQUE (candidato_id, numero_ordem)
);

CREATE TABLE tse_arquivo_importado (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    caminho_relativo TEXT,
    nome            TEXT NOT NULL,
    tipo            TEXT,
    uf              TEXT,
    ano             INTEGER,
    total_registros INTEGER NOT NULL DEFAULT 0,
    hash_sha256     TEXT,
    criado_em       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tse_arquivo_importado_nome_uq UNIQUE (nome)
);

CREATE TABLE tse_meta (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ano                        INTEGER NOT NULL,
    uf                         TEXT NOT NULL,
    dataset                    TEXT NOT NULL,
    status                     TEXT NOT NULL,
    total_registros            INTEGER,
    arquivos_processados       INTEGER NOT NULL DEFAULT 0,
    pagina_erro                INTEGER,
    atualizado_em              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tse_meta_scope_uq UNIQUE (ano, uf, dataset)
);

CREATE INDEX idx_tse_candidato_eleicao_id ON tse_candidato (eleicao_id);
CREATE INDEX idx_tse_candidato_partido_id ON tse_candidato (partido_id);
CREATE INDEX idx_tse_prestacao_eleicao_id ON tse_prestacao_contas (eleicao_id);
CREATE INDEX idx_tse_prestacao_candidato_id ON tse_prestacao_contas (candidato_id);
CREATE INDEX idx_tse_prestacao_partido_id ON tse_prestacao_contas (partido_id);
CREATE INDEX idx_tse_despesa_candidato_prestacao_id ON tse_despesa_candidato (prestacao_contas_id);
CREATE INDEX idx_tse_despesa_candidato_candidato_id ON tse_despesa_candidato (candidato_id);
CREATE INDEX idx_tse_receita_candidato_prestacao_id ON tse_receita_candidato (prestacao_contas_id);
CREATE INDEX idx_tse_receita_candidato_candidato_id ON tse_receita_candidato (candidato_id);
CREATE INDEX idx_tse_receita_candidato_sq_receita ON tse_receita_candidato (sq_receita);
CREATE INDEX idx_tse_receita_orgao_prestacao_id ON tse_receita_orgao_partidario (prestacao_contas_id);
CREATE INDEX idx_tse_receita_orgao_sq_receita ON tse_receita_orgao_partidario (sq_receita);
