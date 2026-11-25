package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"rptgo/internal/report"
)

// Compara o custo de ler 200.000 registros nas três fontes de dados (só a leitura, sem o
// motor): go test -run '^$' -bench Entrada -benchmem ./cmd/rptgo
const registrosBench = 200000

func geraFontesBench() (delimitado, csvTxt, jsonTxt string, cols []string) {
	r := rand.New(rand.NewSource(20261124))
	regioes := []string{"Norte", "Sul", "Leste", "Oeste", "Centro"}
	var d, c, j strings.Builder
	c.WriteString("regiao,vendedor,data_venda,valor\n")
	j.WriteString("[\n")
	for i := 0; i < registrosBench; i++ {
		reg := regioes[r.Intn(len(regioes))]
		ven := fmt.Sprintf("Vendedor %04d", r.Intn(5000))
		dt := fmt.Sprintf("2026-%02d-%02d", 1+r.Intn(12), 1+r.Intn(28))
		val := fmt.Sprintf("%d.%02d", r.Intn(100000), r.Intn(100))
		fmt.Fprintf(&d, "%s|%s|%s|%s\n", reg, ven, dt, val)
		fmt.Fprintf(&c, "%s,%s,%s,%s\n", reg, ven, dt, val)
		sep := ","
		if i == registrosBench-1 {
			sep = ""
		}
		fmt.Fprintf(&j, "  {\"regiao\": %q, \"vendedor\": %q, \"data_venda\": %q, \"valor\": %s}%s\n", reg, ven, dt, val, sep)
	}
	j.WriteString("]\n")
	return d.String(), c.String(), j.String(), []string{"regiao", "vendedor", "data_venda", "valor"}
}

func BenchmarkEntradaDelimitada(b *testing.B) {
	d, _, _, _ := geraFontesBench()
	b.ReportAllocs()
	b.SetBytes(int64(len(d)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leRegistros(d, "|", "regiao", "auto", 4)
	}
}

func BenchmarkEntradaCSV(b *testing.B) {
	_, c, _, _ := geraFontesBench()
	b.ReportAllocs()
	b.SetBytes(int64(len(c)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := leRegistrosCSV(c, detectaDelimitadorCSV(c, 4), "regiao", "auto"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEntradaJSON(b *testing.B) {
	_, _, j, cols := geraFontesBench()
	b.ReportAllocs()
	b.SetBytes(int64(len(j)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := leRegistrosJSON(j, cols); err != nil {
			b.Fatal(err)
		}
	}
}

// Relatório completo (leitura + conversão + ordenação + quebras + paginação) sobre cada fonte.
func BenchmarkRelatorioCompleto(b *testing.B) {
	d, c, j, cols := geraFontesBench()
	toks, _ := report.Tokenize("input\n regiao [string 10],\n vendedor [string 14],\n data_venda [ldate 10],\n valor [amount 10]\nsort regiao\ndetail\n print regiao, valor\nafter regiao\n print total(valor)\nend\n", false)
	s, err := report.Parse(toks)
	if err != nil {
		b.Fatal(err)
	}
	rodar := func(b *testing.B, le func() ([][]string, error)) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			regs, err := le()
			if err != nil {
				b.Fatal(err)
			}
			it, err := report.NovoInterp(s, regs, false)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := it.Run(); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.Run("delimitado", func(b *testing.B) {
		rodar(b, func() ([][]string, error) { return leRegistros(d, "|", "regiao", "auto", 4), nil })
	})
	b.Run("csv", func(b *testing.B) {
		rodar(b, func() ([][]string, error) { return leRegistrosCSV(c, ',', "regiao", "auto") })
	})
	b.Run("json", func(b *testing.B) { rodar(b, func() ([][]string, error) { return leRegistrosJSON(j, cols) }) })
}
