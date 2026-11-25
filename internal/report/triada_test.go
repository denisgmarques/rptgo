package report

import (
	"os"
	"testing"
)

// TRIADSEP e RADIXSEP controlam os separadores de milhar e decimal na saída das máscaras.
func TestTriadSepERadixSep(t *testing.T) {
	casos := []struct{ triad, radix, want string }{
		{"", "", "54,390.00"},   // padrão: vírgula de milhar, ponto decimal
		{".", ",", "54.390,00"}, // formato brasileiro
		{" ", ".", "54 390.00"}, // espaço
		{"'", ",", "54'390,00"},
	}
	for _, c := range casos {
		for k, v := range map[string]string{"TRIADSEP": c.triad, "RADIXSEP": c.radix} {
			if v == "" {
				t.Setenv(k, "")
				os.Unsetenv(k)
			} else {
				t.Setenv(k, v)
			}
		}
		got, err := rodar(t, "input\n  v [amount 10]\ndetail\n  print v using '###,##&.&&'\nend\n", []string{"54390"})
		if err != nil || !igual(espacosUnicos(got), []string{c.want}) && !igual(got, []string{c.want}) {
			t.Errorf("TRIADSEP=%q RADIXSEP=%q: got %q err=%v, want %q", c.triad, c.radix, got, err, c.want)
		}
	}
	// o separador de milhar só aparece quando há dígito à esquerda
	t.Setenv("TRIADSEP", ".")
	os.Unsetenv("RADIXSEP")
	got, _ := rodar(t, "input\n  v [amount 10]\ndetail\n  print v using '###,##&.&&'\nend\n", []string{"123"})
	if !igual(got, []string{"123.00"}) {
		t.Errorf("sem milhar: got %q", got)
	}
}
