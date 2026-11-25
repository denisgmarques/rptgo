package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// dateFmt lê a variável de configuração DATEFMT (padrão DD/MM/YY, o mesmo valor que esse
// sistema usa quando a instalação não define outro).
func dateFmt() string {
	return strings.ToUpper(cfg("DATEFMT", "DD/MM/YY"))
}

var reDataTexto = regexp.MustCompile(`^(\d{1,4})[/.\-](\d{1,2})[/.\-](\d{1,4})$`)
var reTokensData = regexp.MustCompile(`YYYY|YY|DD|MM`)

// ParseData interpreta um texto de data conforme o DATEFMT configurado (ordem dos campos) e
// o CENTURY_CUTOFF (ano de 2 dígitos: >= corte vira 19xx, abaixo vira 20xx). Um ano já escrito
// com 4 dígitos no texto é usado ao pé da letra, sem corte de século.
func ParseData(txt string) (time.Time, bool, error) {
	txt = strings.TrimSpace(txt)
	if txt == "" {
		return time.Time{}, false, nil
	}
	m := reDataTexto.FindStringSubmatch(txt)
	if m == nil {
		return time.Time{}, false, Erro("data inválida: %q", txt)
	}
	a, b, c := m[1], m[2], m[3]
	f := dateFmt()
	ordem := reTokensData.FindAllString(f, -1)
	campos := map[string]string{}
	for i, tok := range ordem {
		switch i {
		case 0:
			campos[tok] = a
		case 1:
			campos[tok] = b
		case 2:
			campos[tok] = c
		}
	}
	diaTxt, mesTxt := campos["DD"], campos["MM"]
	if diaTxt == "" {
		diaTxt = "1"
	}
	if mesTxt == "" {
		mesTxt = "1"
	}
	anoTxt := campos["YYYY"]
	if anoTxt == "" {
		anoTxt = campos["YY"]
	}
	if len(a) == 4 && (len(ordem) == 0 || (ordem[0] != "YYYY" && ordem[0] != "YY")) {
		// AAAA-MM-DD (ISO) também é aceito, qualquer que seja o DATEFMT, desde que o formato
		// configurado não comece pelo ano (nesse caso o primeiro campo já é o ano).
		anoTxt, mesTxt, diaTxt = a, b, c
	}
	dia, _ := strconv.Atoi(diaTxt)
	mes, _ := strconv.Atoi(mesTxt)
	ano, _ := strconv.Atoi(anoTxt)
	if len(anoTxt) <= 2 {
		corte, _ := strconv.Atoi(cfg("CENTURY_CUTOFF", "30"))
		if ano >= corte {
			ano += 1900
		} else {
			ano += 2000
		}
	}
	return time.Date(ano, time.Month(mes), dia, 0, 0, 0, 0, time.UTC), true, nil
}

// FmtData formata uma data pra impressão. A largura do ano (2 ou 4 dígitos) depende do TIPO
// da coluna (date=2, ldate=4 - são tipos distintos), não do texto literal
// configurado em DATEFMT - por isso aqui sempre força a largura certa.
func FmtData(d time.Time, valido bool, longa bool) string {
	if !valido {
		return ""
	}
	f := dateFmt()
	var ano string
	if longa {
		ano = fmt.Sprintf("%04d", d.Year())
	} else {
		ano = fmt.Sprintf("%02d", d.Year()%100)
	}
	f = strings.ReplaceAll(f, "YYYY", ano)
	f = strings.ReplaceAll(f, "YY", ano)
	f = strings.ReplaceAll(f, "DD", fmt.Sprintf("%02d", d.Day()))
	f = strings.ReplaceAll(f, "MM", fmt.Sprintf("%02d", int(d.Month())))
	return f
}

// Hora é uma hora do dia (tipo time do RPT): segundos desde a meia-noite, de 0 a 86399. Não é
// um intervalo e não é negativa; a aritmética dá a volta em 24h.
type Hora int64

var reHora = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?$`)

// ParseHora lê H:MM, HH:MM ou HH:MM:SS (fração de segundo é ignorada). O texto vazio é nulo
// (ok=false); fora de 0-23:0-59:0-59 é erro.
func ParseHora(txt string) (Hora, bool, error) {
	txt = strings.TrimSpace(txt)
	if txt == "" {
		return 0, false, nil
	}
	m := reHora.FindStringSubmatch(txt)
	if m == nil {
		return 0, false, Erro("hora inválida: %q (use HH:MM)", txt)
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	se := 0
	if m[3] != "" {
		se, _ = strconv.Atoi(m[3])
	}
	if h > 23 || mi > 59 || se > 59 {
		return 0, false, Erro("hora inválida: %q (horas 0-23, minutos e segundos 0-59)", txt)
	}
	return Hora(h*3600 + mi*60 + se), true, nil
}

// ISO devolve a hora em ISO-8601 (HH:MM:SS), usada nas saídas CSV e JSON.
func (h Hora) ISO() string {
	return fmt.Sprintf("%02d:%02d:%02d", int(h)/3600, int(h)%3600/60, int(h)%60)
}

// FmtHora formata uma hora conforme TIMEFMT (padrão HH:MM; SS também é aceito).
func FmtHora(h Hora) string {
	f := cfg("TIMEFMT", "HH:MM")
	f = strings.ReplaceAll(f, "HH", fmt.Sprintf("%02d", int(h)/3600))
	f = strings.ReplaceAll(f, "MM", fmt.Sprintf("%02d", int(h)%3600/60))
	f = strings.ReplaceAll(f, "SS", fmt.Sprintf("%02d", int(h)%60))
	return f
}

// horaDoDia devolve a hora do relógio do relatório como Hora.
func horaDoDia(agora time.Time) Hora {
	return Hora(agora.Hour()*3600 + agora.Minute()*60 + agora.Second())
}
