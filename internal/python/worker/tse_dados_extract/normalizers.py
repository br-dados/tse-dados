"""Normalizacao de valores por convencao de campo (espelha parse/tipos.go)."""

import re

NULOS = {"", "#NULO", "#NULO#", "#NE", "NÃO DIVULGÁVEL", "NAO DIVULGAVEL"}

DATE_RE = re.compile(r"^\d{2}/\d{2}/\d{4}$")


def texto(valor):
    s = "" if valor is None else str(valor).strip()
    return "" if s.upper() in NULOS else s


def decimal(valor):
    """'1.234,56' -> '1234.56'. Datetime/float -> string decimal simples."""
    s = texto(valor)
    if not s:
        return ""
    if isinstance(valor, float):
        return "%.2f" % valor
    s = s.replace(".", "").replace(",", ".")
    try:
        return "%.2f" % float(s)
    except ValueError:
        return s


def _para_dmY(valor):
    import datetime
    try:
        # datetime do pandas/openpyxl -> dd/mm/aaaa
        if hasattr(valor, "strftime"):
            return valor.strftime("%d/%m/%Y")
        s = str(valor).strip()
        if DATE_RE.match(s):
            return s
        for fmt in ("%Y-%m-%d", "%Y/%m/%d", "%d-%m-%Y", "%Y%m%d"):
            try:
                d = datetime.datetime.strptime(s, fmt)
                return d.strftime("%d/%m/%Y")
            except ValueError:
                continue
        return s
    except Exception:
        return ""


def data(valor):
    s = texto(valor)
    if not s:
        return ""
    return _para_dmY(valor)


def documento(valor):
    """CPF/CNPJ: apenas digitos; placeholders -> vazio."""
    s = texto(valor)
    if not s:
        return ""
    if s in ("-4", "-1", "00000000000", "00000000000000"):
        return ""
    d = re.sub(r"\D", "", s)
    if not d or not d.strip("0"):
        return ""
    return d


_CAMPO_DECIMAL = {"valor", "valor_despesa", "valor_receita", "vr_despesa", "vr_receita", "vr_bem"}
_CAMPO_DOC = {"cpf", "cpf_cnpj", "documento_doador", "cnpj_prestador_conta", "numero_cpf"}


def normalizar(field, valor):
    """Normaliza um valor bruto conforme a convencao do campo."""
    if valor is None:
        return ""
    campo = str(field).lower()
    if campo in _CAMPO_DECIMAL or campo.startswith("valor") or campo.endswith("_valor"):
        return decimal(valor)
    if campo in _CAMPO_DOC or "documento" in campo or campo in ("cpf", "cnpj"):
        return documento(valor)
    if "data" in campo or campo.startswith("dt_") or campo.endswith("_data"):
        return data(valor)
    if campo == "id":
        return ""
    return texto(valor)
