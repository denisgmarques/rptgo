package report

import (
	"os"
	"strings"
	"testing"
)

func TestParseHora(t *testing.T) {
	casos := []struct {
		txt  string
		want string // ISO, "" = nulo, "erro" = erro
	}{
		{"14:30", "14:30:00"}, {"9:05", "09:05:00"}, {"00:00", "00:00:00"}, {"23:59", "23:59:00"},
		{"08:05:09", "08:05:09"}, {"14:30:00.250", "14:30:00"}, {"  14:30 ", "14:30:00"},
		{"", ""}, {"   ", ""},
		{"24:00", "erro"}, {"25:10", "erro"}, {"12:60", "erro"}, {"12:30:61", "erro"},
		{"abc", "erro"}, {"12", "erro"}, {"12:3", "erro"}, {"-1:30", "erro"}, {"2026-03-15 14:30:00", "erro"},
	}
	for _, c := range casos {
		h, ok, err := ParseHora(c.txt)
		switch c.want {
		case "erro":
			if err == nil {
				t.Errorf("%q: esperava erro", c.txt)
			}
		case "":
			if err != nil || ok {
				t.Errorf("%q: esperava nulo, veio ok=%v err=%v", c.txt, ok, err)
			}
		default:
			if err != nil || !ok || h.ISO() != c.want {
				t.Errorf("%q: got %v ok=%v err=%v, want %s", c.txt, h.ISO(), ok, err, c.want)
			}
		}
	}
}

func TestFmtHoraComTIMEFMT(t *testing.T) {
	h := Hora(8*3600 + 5*60 + 9)
	for fmtEnv, want := range map[string]string{"": "08:05", "HH:MM": "08:05", "HH.MM": "08.05", "HH:MM:SS": "08:05:09", "HHhMM": "08h05"} {
		if fmtEnv == "" {
			t.Setenv("TIMEFMT", "")
			os.Unsetenv("TIMEFMT")
		} else {
			t.Setenv("TIMEFMT", fmtEnv)
		}
		if got := FmtHora(h); got != want {
			t.Errorf("TIMEFMT=%q: got %q, want %q", fmtEnv, got, want)
		}
	}
}

const scriptHora = "input\n  id [numeric],\n  h [time],\n  g [time]\n"

func TestColunaTime(t *testing.T) {
	got, err := rodar(t, scriptHora+"sort h\ndetail\n  print id, h, g\nend\n",
		[]string{"1|14:30|08:00", "2|9:05|", "3||23:59:30", "4|00:00|12:00"})
	if err != nil {
		t.Fatal(err)
	}
	// nulos vêm primeiro na ordenação; a hora imprime como HH:MM; vazio imprime em branco
	want := []string{"3 23:59", "4 00:00 12:00", "2 09:05", "1 14:30 08:00"}
	if g := espacosUnicos(got); !igual(g, want) {
		t.Errorf("got %q, want %q", g, want)
	}
}

func TestHoraOperacoes(t *testing.T) {
	script := scriptHora + "detail\n  print %s\nend\n"
	ok := []struct{ expr, want string }{
		{"1:00 + 60", "02:00"},
		{"23:30 + 60", "00:30"},
		{"00:10 - 20", "23:50"},
		{"60 + 1:00", "02:00"},
		{"10:30 - 9:45", "45"},
		{"h + 90", "16:00"},
		{"h - g", "390"},
		{"g - h", "-390"},
		{"h > g", "1"},
		{"h < g", "0"},
		{"h >= 14:30", "1"},
		{"h = 14:30", "1"},
		{"h <> 14:30", "0"},
		{"h is null", "0"},
	}
	for _, c := range ok {
		got, err := rodar(t, strings.Replace(script, "%s", c.expr, 1), []string{"1|14:30|08:00"})
		if err != nil || !igual(espacosUnicos(got), []string{c.want}) {
			t.Errorf("%s: got %q err=%v, want %q", c.expr, got, err, c.want)
		}
	}
	erros := []struct{ expr, quer string }{
		{"h + g", "operação inválida com horas"},
		{"h * 2", "operação inválida com horas"},
		{"h + 'x'", "operação inválida com horas"},
		{"h = 5", "comparar uma hora com outro tipo"},
		{"h < 'x'", "comparar uma hora com outro tipo"},
	}
	for _, c := range erros {
		_, err := rodar(t, strings.Replace(script, "%s", c.expr, 1), []string{"1|14:30|08:00"})
		if err == nil || !strings.Contains(err.Error(), c.quer) {
			t.Errorf("%s: err = %v, quer %q", c.expr, err, c.quer)
		}
	}
}

func TestColunaTimeInvalida(t *testing.T) {
	for _, d := range []string{"1|25:00|08:00", "1|12:60|08:00", "1|abc|08:00", "1|2026-03-15 14:30:00|08:00"} {
		_, err := rodar(t, scriptHora+"detail\n  print id\nend\n", []string{d})
		if err == nil || !strings.Contains(err.Error(), "valor inválido em 'h'") || !strings.Contains(err.Error(), "hora inválida") {
			t.Errorf("%q: err = %v", d, err)
		}
	}
}

func TestHourContinuaImprimindoComoAntes(t *testing.T) {
	t.Setenv("RPT_NOW", "2026-11-24 17:47:30")
	got, err := rodar(t, "input\n  id [numeric]\ndetail\n  print hour, hour - 17:00, hour > 12:00\nend\n", []string{"1"})
	if err != nil || !igual(espacosUnicos(got), []string{"17:47 47 1"}) {
		t.Errorf("got %q err=%v", got, err)
	}
}

func TestTimeConstanteInvalidaNoScript(t *testing.T) {
	_, err := rodar(t, "input\n  id [numeric]\ndetail\n  print 25:00\nend\n", []string{"1"})
	if err == nil || !strings.Contains(err.Error(), "hora inválida") {
		t.Errorf("err = %v", err)
	}
}
