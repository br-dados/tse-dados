ALTER TABLE tse_arquivo_importado DROP CONSTRAINT IF EXISTS tse_arquivo_importado_nome_uf_uq;
ALTER TABLE tse_arquivo_importado ADD CONSTRAINT tse_arquivo_importado_nome_uq UNIQUE (nome);
