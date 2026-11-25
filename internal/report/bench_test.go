package report

import (
	"runtime"
	"testing"
)

// BenchmarkConverte200k mede a conversão de 200.000 registros em Row (RNF-01).
//
// Como reproduzir a comparação antes/depois da troca de representação de Row:
//
//	go test -run '^$' -bench Converte200k -benchmem ./internal/report/
//
// rode uma vez no commit anterior à troca de Row (map[string]Value) e uma vez depois
// (Row = []Value). Compare B/op, allocs/op e a métrica retido-B (heap retido com todos os
// registros vivos, medido com runtime.MemStats depois de um GC). Qualquer redução serve.
func BenchmarkConverte200k(b *testing.B) {
	s, err := scriptVendas200kParseado()
	if err != nil {
		b.Fatal(err)
	}
	linhas := geraRegistros200k(total200k)
	b.ReportAllocs()
	b.ResetTimer()
	var retido uint64
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runtime.GC()
		var antes, depois runtime.MemStats
		runtime.ReadMemStats(&antes)
		b.StartTimer()

		it, err := NovoInterp(s, linhas, false)
		if err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&depois)
		runtime.KeepAlive(it)
		if depois.HeapAlloc > antes.HeapAlloc {
			retido = depois.HeapAlloc - antes.HeapAlloc
		}
		b.StartTimer()
	}
	b.ReportMetric(float64(retido), "retido-B")
}
