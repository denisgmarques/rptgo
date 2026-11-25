package report

import (
	"os"
	"testing"
	"time"
)

func TestParseDataAnoQuatroDigitos(t *testing.T) {
	casos := []struct {
		fmt, txt string
		want     string // AAAA-MM-DD
	}{
		{"", "05/03/1985", "1985-03-05"}, // padrão DD/MM/YY com ano de 4 dígitos
		{"", "31/12/29", "2029-12-31"},   // ano de 2 dígitos usa o CENTURY_CUTOFF
		{"", "01/02/1700", "1700-02-01"}, // sem corte de século quando há 4 dígitos
		{"", "2020-01-15", "2020-01-15"}, // ISO com o formato padrão
		{"MM/DD/YYYY", "03/05/1985", "1985-03-05"},
		{"MM/DD/YYYY", "2020-01-15", "2020-01-15"}, // ISO continua valendo com DATEFMT de mês primeiro
		{"MM/DD/YY", "12/31/2099", "2099-12-31"},
		{"MM/DD/YY", "2020-01-15", "2020-01-15"},
		{"YYYY/MM/DD", "2020/01/15", "2020-01-15"}, // formato que começa pelo ano
		{"YYYY/MM/DD", "2020-01-15", "2020-01-15"},
		{"YY/MM/DD", "85/03/05", "1985-03-05"},
	}
	for _, c := range casos {
		if c.fmt == "" {
			t.Setenv("DATEFMT", "")
			os.Unsetenv("DATEFMT")
		} else {
			t.Setenv("DATEFMT", c.fmt)
		}
		d, ok, err := ParseData(c.txt)
		if err != nil || !ok {
			t.Errorf("DATEFMT=%q %q: ok=%v err=%v", c.fmt, c.txt, ok, err)
			continue
		}
		if got := d.Format(time.DateOnly); got != c.want {
			t.Errorf("DATEFMT=%q %q: got %s, want %s", c.fmt, c.txt, got, c.want)
		}
	}
}
