#!/usr/bin/env bash
# Regenerates the PNG previews used in README.md (docs/images/*.png).
# Needs: Go, Google Chrome (or Chromium) and poppler-utils (pdftoppm).
# The clock is pinned with RPT_NOW so the images never change between runs.
set -euo pipefail
cd "$(dirname "$0")/.."

RPT="$(mktemp -d)/rptgo"
go build -o "$RPT" ./cmd/rptgo
export RPT_NOW="2026-06-30 09:00"
CHROME="$(command -v google-chrome || command -v chromium || command -v chromium-browser)"
TMP="$(mktemp -d)"

# png NAME  -> converts $TMP/NAME.html to a one-page PDF, then to docs/images/NAME.png
png() {
  "$CHROME" --headless=new --no-sandbox --disable-gpu --no-pdf-header-footer \
    --print-to-pdf="$TMP/$1.pdf" "file://$TMP/$1.html" >/dev/null 2>&1
  pdftoppm -f 1 -l 1 -r 80 -png -singlefile "$TMP/$1.pdf" "docs/images/$1"
}

# 1. Invoice: ASCII boxes -> line characters, custom template, logo and variables.
"$RPT" --format html --box \
  --template examples/invoice-html/template.html --logo examples/invoice-html/logo.svg \
  --var Company="Northwind Ltd" --var Invoice=2026-0142 \
  --var BillTo="Acme Corp, 100 Main St, Springfield" --var Footer="payment due in 30 days" \
  examples/invoice-html/report.rp examples/invoice-html/data.txt > "$TMP/invoice.html"
png invoice

# 2. Purchase orders: customer > order breaks with the built-in A4 template.
"$RPT" --format html --title "Purchase orders" \
  examples/purchase-orders/report.rp examples/purchase-orders/data.txt > "$TMP/purchase-orders.html"
png purchase-orders

# 3. Payroll: per-department totals with the built-in template.
"$RPT" --format html --title "Payroll by department" \
  examples/payroll-by-department/report.rp examples/payroll-by-department/data.txt > "$TMP/payroll.html"
png payroll
echo "wrote docs/images/{invoice,purchase-orders,payroll}.png"
