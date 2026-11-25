package report

import "testing"

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
