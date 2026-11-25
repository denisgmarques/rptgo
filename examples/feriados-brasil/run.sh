#!/usr/bin/env bash
# Uma chamada com curl, e o JSON que volta entra direto no relatorio.
#   ./run.sh          # feriados do ano 2026
#   ./run.sh 2027
set -euo pipefail
cd "$(dirname "$0")"
RPT="${RPT:-$(cd ../.. && pwd)/rptgo}"
[ -x "$RPT" ] || RPT="$(command -v rpt || command -v rptgo)"
curl -sf "https://brasilapi.com.br/api/feriados/v1/${1:-2026}" | "$RPT" --input json relatorio.rp -
