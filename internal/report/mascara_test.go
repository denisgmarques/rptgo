package report

import (
	"strings"
	"testing"
)

// Quando o número tem mais dígitos do que a máscara comporta, os dígitos de ordem mais alta que
// sobram são impressos inteiros, à esquerda e na ordem certa (não são cortados nem invertidos).
func TestMascaraComDigitosAMais(t *testing.T) {
	casos := []struct{ mascara, valor, want string }{
		{"####", "123457", "123457"},
		{"&&&&", "123456.789", "123457"},
		{"#,###.##", "123456.789", "123,456.79"},
		{"###,##&.&&", "1234567.891", "1234,567.89"}, // o excedente sai cru, sem separador de milhar
		{"&&", "98765", "98765"},
		{"&&&&", "42", "0042"},
	}
	for _, c := range casos {
		got, err := rodar(t, "input\n  v [float]\nleft margin 0\ndetail\n  print v using '"+c.mascara+"'\nend\n", []string{c.valor})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("máscara %q com %s: got %q err=%v, want %q", c.mascara, c.valor, got, err, c.want)
		}
	}
}

// Com o sinal (ou a moeda) repetido na máscara, o símbolo flutua e fica colado no primeiro
// dígito, inclusive quando ele cairia na posição de uma vírgula. Um sinal único fica fixo.
func TestMascaraSinalFlutuante(t *testing.T) {
	casos := []struct{ mascara, valor, want string }{
		{"--,---,--&.&&", "-1", "-1.00"},
		{"--,---,--&.&&", "-123", "-123.00"},
		{"--,---,--&.&&", "-1234", "-1,234.00"},
		{"--,---,--&.&&", "-123456", "-123,456.00"},
		{"--,---,--&.&&", "123", "123.00"},
		{"++,+++,++&.&&", "123", "+123.00"},
		{"$$,$$$,$$&.&&", "123456", "$123,456.00"},
		{"((,(((,((&.&&)", "-123", "(123.00)"},
		{"-###,##&.&&", "-123", "-    123.00"},
	}
	for _, c := range casos {
		got, err := rodar(t, "input\n  v [float]\nleft margin 0\ndetail\n  print v using '"+c.mascara+"'\nend\n", []string{c.valor})
		// os brancos à esquerda dependem só do tamanho da máscara
		for i := range got {
			got[i] = strings.TrimSpace(got[i])
		}
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("máscara %q com %s: got %q err=%v, want %q", c.mascara, c.valor, got, err, c.want)
		}
	}
}
