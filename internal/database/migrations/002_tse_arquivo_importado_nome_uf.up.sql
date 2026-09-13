-- Um zip nacional pode alimentar varias UFs: a deduplicacao de arquivos
-- importados passa a ser por (nome, uf) em vez de apenas (nome).
ALTER TABLE tse_arquivo_importado DROP CONSTRAINT IF EXISTS tse_arquivo_importado_nome_uq;
ALTER TABLE tse_arquivo_importado ADD CONSTRAINT tse_arquivo_importado_nome_uf_uq UNIQUE (nome, uf);
