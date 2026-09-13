"""Wrapper de linha de comando do worker.

Uso: python -m tse_dados_extract --manifest <manifest.json>
Le o manifest e emite CSVs normalizados em <manifest.output_dir>/<dataset>/<target>.csv.
Progresso e emitido como linhas JSON no stdout.

O TSE publica um zip nacional por (grupo, ano) com 1 CSV por UF. Cada arquivo do
manifest traz os grupos (tipos) a extrair, filtrando os CSVs internos por
prefixo+ano e por UF (quando informada). Agregados `_BR`/`_BRASIL` sao ignorados.
"""

import argparse
import csv
import json
import os
import sys
import zipfile

from . import normalizers, readers


def _constante(field, table):
    if field == "tipo_prestador":
        return table.get("prestador", "")
    if field == "tipo_registro":
        return table.get("tipo_registro", "")
    return ""


def _valor(linha, col, table):
    header = col.get("header", "")
    field = col.get("field", "")
    if header:
        return linha.get(header, "")
    return _constante(field, table)


def _abrir_grupo(escritores, grupo, output_dir):
    dataset = grupo["dataset"]
    if dataset in escritores:
        return escritores[dataset]
    destinos = {}
    for table in grupo.get("tables", []):
        target = table["target"]
        cols = table.get("columns", [])
        out_path = os.path.join(output_dir, dataset, target + ".csv")
        os.makedirs(os.path.dirname(out_path), exist_ok=True)
        fo = open(out_path, "w", newline="", encoding="utf-8")
        writer = csv.writer(fo)
        writer.writerow([c["field"] for c in cols])
        destinos[target] = (fo, writer, cols)
    escritores[dataset] = destinos
    return destinos


def _escrever(destinos, grupo, linha, contagem):
    dataset = grupo["dataset"]
    for table in grupo.get("tables", []):
        target = table["target"]
        fo, writer, cols = destinos[target]
        writer.writerow([normalizers.normalizar(c["field"], _valor(linha, c, table)) for c in cols])
        contagem[(dataset, target)] = contagem.get((dataset, target), 0) + 1


def _grupo_para(nome, ano, grupos):
    for grupo in grupos:
        if readers.casa_grupo(nome, grupo.get("prefijo", ""), ano):
            return grupo
    return None


def main(argv=None):
    parser = argparse.ArgumentParser(prog="tse_dados_extract")
    parser.add_argument("--manifest", required=True, help="caminho do manifest.json")
    args = parser.parse_args(argv)

    with open(args.manifest, encoding="utf-8") as f:
        manifest = json.load(f)

    output_dir = manifest.get("output_dir", ".")

    for arq in manifest.get("files", []):
        path = arq["path"]
        formato = arq.get("format", "zip")
        ano = arq.get("ano", 0)
        ufs = {u.upper() for u in arq.get("ufs", [])}
        grupos = arq.get("grupos", [])
        escritores = {}
        contagem = {}

        try:
            if formato == "zip":
                with zipfile.ZipFile(path) as z:
                    for nome in z.namelist():
                        base = os.path.basename(nome)
                        if not base.lower().endswith((".csv", ".xlsx")):
                            continue
                        grupo = _grupo_para(base, ano, grupos)
                        if grupo is None:
                            continue
                        uf = readers.uf_do_nome(base)
                        if uf in readers.AGREGADOS:
                            continue
                        if ufs and uf not in ufs:
                            continue
                        destinos = _abrir_grupo(escritores, grupo, output_dir)
                        for linha in readers.pacientes_zip(z, nome):
                            _escrever(destinos, grupo, linha, contagem)
            else:
                grupo = grupos[0]
                destinos = _abrir_grupo(escritores, grupo, output_dir)
                for linha in readers.pacientes(path, formato):
                    _escrever(destinos, grupo, linha, contagem)
        except Exception as e:  # noqa: BLE001
            for destinos in escritores.values():
                for fo, _w, _c in destinos.values():
                    fo.close()
            print(json.dumps({
                "estagio": "extract",
                "dataset": grupos[0]["dataset"] if grupos else "",
                "arquivo": os.path.basename(path),
                "erro": str(e),
            }), flush=True)
            sys.exit(1)
        finally:
            for destinos in escritores.values():
                for fo, _w, _c in destinos.values():
                    fo.close()

        for grupo in grupos:
            dataset = grupo["dataset"]
            for table in grupo.get("tables", []):
                target = table["target"]
                print(json.dumps({
                    "estagio": "extract",
                    "dataset": dataset,
                    "arquivo": os.path.basename(path),
                    "target": target,
                    "registros": contagem.get((dataset, target), 0),
                }), flush=True)
        sys.stderr.write("processado %s: %d registros\n" % (os.path.basename(path), sum(contagem.values())))

    sys.stderr.write("ok\n")


if __name__ == "__main__":
    main()
