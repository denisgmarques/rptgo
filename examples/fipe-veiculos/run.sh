#!/usr/bin/env bash
# Tabela FIPE de uma marca e de alguns modelos, direto da API para o rptgo (sem jq):
# o JSON que o curl traz entra como esta no relatorio.
#
#   ./run.sh                    # marca 21 e modelos 437 e 7097 (poucas chamadas)
#   ./run.sh 21 437 6885        # marca 21 e os modelos 437 e 6885
#   FIPE_API=https://... ./run.sh
#
# Os codigos de marca e de modelo vem de .../cars/brands e .../cars/brands/<marca>/models.
# A API aberta tem limite de chamadas por dia: cada modelo custa 1 chamada mais 1 por ano.
set -euo pipefail
cd "$(dirname "$0")"
RPT="${RPT:-$(cd ../.. && pwd)/rptgo}"
[ -x "$RPT" ] || RPT="$(command -v rpt || command -v rptgo)"
API="${FIPE_API:-https://fipe.parallelum.com.br/api/v2/cars/brands}"

MARCA="${1:-21}"
[ $# -gt 0 ] && shift
MODELOS=("$@")
[ ${#MODELOS[@]} -gt 0 ] || MODELOS=(437 7097)

for modelo in "${MODELOS[@]}"; do
  # lista de anos do modelo -> um codigo por linha (o proprio rptgo le o JSON)
  curl -sf "$API/$MARCA/models/$modelo/years" |
    "$RPT" --input json --format csv --csv-no-header anos.rp - |
    while read -r ano; do
      curl -sf "$API/$MARCA/models/$modelo/years/$ano"
      echo
    done
done | "$RPT" --input json relatorio.rp -
