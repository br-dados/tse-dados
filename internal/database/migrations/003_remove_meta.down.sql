CREATE TABLE IF NOT EXISTS tse_meta (
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
