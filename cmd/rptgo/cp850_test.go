package main

import "testing"

func TestCP850Tabela256(t *testing.T) {
	vistos := map[rune]int{}
	for b, r := range cp850ParaRune {
		if r < 0 || r > 0x10FFFF {
			t.Errorf("byte 0x%02X: code point inválido %U", b, r)
		}
		if outro, ok := vistos[r]; ok {
			t.Errorf("code point %U repetido nos bytes 0x%02X e 0x%02X", r, outro, b)
		}
		vistos[r] = b
	}
	if len(vistos) != 256 {
		t.Errorf("esperadas 256 entradas distintas, vieram %d", len(vistos))
	}
	// Âncoras conferidas à mão contra a tabela pública (Unicode CP850.TXT).
	ancoras := map[byte]rune{
		0x00: 0x0000, 0x41: 'A', 0x7E: '~', 0x80: 'Ç', 0x81: 'ü', 0x82: 'é', 0x87: 'ç',
		0x9B: 'ø', 0x9C: '£', 0x9F: 'ƒ', 0xA0: 'á', 0xA4: 'ñ', 0xA7: 'º', 0xAD: '¡',
		0xB0: '░', 0xC4: '─', 0xC6: 'ã', 0xC7: 'Ã', 0xD0: 'ð', 0xD5: 'ı', 0xE1: 'ß', 0xE7: 'þ',
		0xF1: '±', 0xF8: '°', 0xFD: '²', 0xFF: 0x00A0,
	}
	for b, want := range ancoras {
		if got := cp850ParaRune[b]; got != want {
			t.Errorf("byte 0x%02X: got %U, want %U", b, got, want)
		}
	}
}

func TestCP850DivergeDeLatin1(t *testing.T) {
	for b := 0x80; b <= 0xFF; b++ {
		if cp850ParaRune[b] == rune(b) {
			t.Errorf("byte 0x%02X decodifica igual em cp850 e latin-1 (%U)", b, rune(b))
		}
		if got, want := decodificar([]byte{byte(b)}, "cp850"), string(cp850ParaRune[b]); got != want {
			t.Errorf("decodificar cp850 0x%02X: got %q, want %q", b, got, want)
		}
		if decodificar([]byte{byte(b)}, "cp850") == decodificar([]byte{byte(b)}, "latin-1") {
			t.Errorf("byte 0x%02X: cp850 e latin-1 deveriam diferir", b)
		}
	}
}

func TestCP850RoundTrip(t *testing.T) {
	for b := 0; b < 256; b++ {
		if got, ok := cp850ParaByte[cp850ParaRune[b]]; !ok || got != byte(b) {
			t.Errorf("byte 0x%02X: tabela inversa devolveu 0x%02X (ok=%v)", b, got, ok)
		}
		in := []byte{byte(b)}
		if out := codificar(decodificar(in, "cp850"), "cp850"); len(out) != 1 || out[0] != byte(b) {
			t.Errorf("round-trip do byte 0x%02X falhou: %v", b, out)
		}
	}
	// todos os 256 bytes de uma vez
	todos := make([]byte, 256)
	for i := range todos {
		todos[i] = byte(i)
	}
	if got := codificar(decodificar(todos, "cp850"), "cp850"); string(got) != string(todos) {
		t.Error("round-trip do bloco completo de 256 bytes falhou")
	}
}
