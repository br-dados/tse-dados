"""Leitores de arquivos do TSE (.xlsx/.csv/.zip) -> iterador de linhas (dict)."""

import csv
import io
import os
import zipfile

# Siglas de UF aceitas (o TSE publica 1 CSV por UF dentro do zip nacional).
UFS = {
    "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS",
    "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC",
    "SP", "SE", "TO",
}

# Agregados nacionais publicados no zip: devem ser descartados.
AGREGADOS = {"BR", "BRASIL"}


def _decodifica(data):
    """Decodifica bytes do TSE (utf-8 quando possivel, senao ISO-8859-1)."""
    try:
        return data.decode("utf-8-sig")
    except UnicodeDecodeError:
        return data.decode("latin-1")


def _read_csv(texto):
    """Le conteudo CSV (TSE: separador ';') como lista de dict (chave = header)."""
    reader = csv.DictReader(io.StringIO(texto), delimiter=";")
    for row in reader:
        yield row


def uf_do_nome(nome):
    """Extrai a UF do nome base de um CSV do TSE ({prefijo}_{ano}_{UF}.csv)."""
    base = os.path.basename(nome)
    partes = base.rsplit(".", 1)[0].split("_")
    if not partes:
        return ""
    return partes[-1].upper()


def casa_grupo(nome, prefijo, ano):
    """Informa se o CSV interno pertence ao grupo (prefixo + ano)."""
    base = os.path.basename(nome).lower()
    return base.startswith(("%s_%d_" % (prefijo, ano)).lower())


def pacientes_zip(zf, nome):
    """Linhas de uma entrada do zip (csv/xlsx)."""
    low = nome.lower()
    if low.endswith(".csv"):
        with zf.open(nome) as f:
            for row in _read_csv(_decodifica(f.read())):
                yield row
    elif low.endswith(".xlsx"):
        with zf.open(nome) as f:
            yield from _read_bytes_xlsx(f.read())


def pacientes(arquivo, formato):
    """Linhas de um arquivo unico (csv/xlsx) ou de todas as entradas de um zip."""
    if formato == "csv":
        with open(arquivo, "rb") as f:
            for row in _read_csv(_decodifica(f.read())):
                yield row
        return
    if formato == "xlsx":
        yield from _read_xlsx(arquivo)
        return
    if formato == "zip":
        with zipfile.ZipFile(arquivo) as z:
            for nome in z.namelist():
                yield from pacientes_zip(z, nome)
        return
    raise ValueError("formato desconhecido: %r" % formato)


def _read_xlsx(path):
    try:
        import pandas as pd
    except ImportError as e:
        raise RuntimeError("dependencia ausente: pandas/openpyxl. Instale com 'pip install pandas openpyxl'") from e
    df = pd.read_excel(path, dtype=str)
    for _, row in df.fillna("").iterrows():
        yield {key: ("" if pd.isna(v) else str(v)) for key, v in row.items()}


def _read_bytes_xlsx(data):
    import io as _io
    try:
        import pandas as pd
    except ImportError as e:
        raise RuntimeError("dependencia ausente: pandas/openpyxl. Instale com 'pip install pandas openpyxl'") from e
    df = pd.read_excel(_io.BytesIO(data), dtype=str)
    for _, row in df.fillna("").iterrows():
        yield {key: ("" if pd.isna(v) else str(v)) for key, v in row.items()}
