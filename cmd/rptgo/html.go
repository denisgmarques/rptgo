package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"rptgo/internal/report"
)

// Saída --format html: cada página do relatório vira um bloco de texto monoespaçado dentro de
// um modelo HTML. O modelo é texto comum com marcadores no estilo {{Nome}}:
//
//	{{Title}}                   valor escapado para HTML
//	{{#Pages}} ... {{/Pages}}   bloco repetido uma vez por página
//
// Variáveis globais: Title, Script, Date, Time, Columns, PageCount, Logo, mais as de --var.
// Dentro do bloco de páginas: PageNumber e Text (o texto da página). Uma variável
// desconhecida é erro, com a linha do modelo, em vez de sair vazia.

var reNomeVar = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var variaveisGlobais = []string{"Title", "Script", "Date", "Time", "Columns", "PageCount", "Logo"}
var variaveisDePagina = []string{"PageNumber", "Text"}

type tokenTpl struct {
	tipo  string // "texto", "var", "inicio", "fim"
	valor string
	linha int
}

// tokenizaTemplate separa o modelo em texto, variáveis e marcadores de bloco.
func tokenizaTemplate(tpl string) ([]tokenTpl, error) {
	var toks []tokenTpl
	linha := 1
	for len(tpl) > 0 {
		i := strings.Index(tpl, "{{")
		if i < 0 {
			toks = append(toks, tokenTpl{"texto", tpl, linha})
			break
		}
		if i > 0 {
			toks = append(toks, tokenTpl{"texto", tpl[:i], linha})
			linha += strings.Count(tpl[:i], "\n")
		}
		tpl = tpl[i+2:]
		j := strings.Index(tpl, "}}")
		if j < 0 {
			return nil, fmt.Errorf("modelo, linha %d: '{{' sem '}}'", linha)
		}
		nome := strings.TrimSpace(tpl[:j])
		tpl = tpl[j+2:]
		switch {
		case nome == "#Pages":
			toks = append(toks, tokenTpl{"inicio", nome, linha})
		case nome == "/Pages":
			toks = append(toks, tokenTpl{"fim", nome, linha})
		case reNomeVar.MatchString(nome):
			toks = append(toks, tokenTpl{"var", nome, linha})
		default:
			return nil, fmt.Errorf("modelo, linha %d: marcador inválido {{%s}}", linha, nome)
		}
	}
	return toks, nil
}

func ehEmLista(lista []string, v string) bool { return contem(lista, v) }

// renderizaTemplate aplica o modelo: globais (já em texto puro) e páginas (texto de cada
// página). Todos os valores são escapados para HTML.
func renderizaTemplate(tpl string, globais map[string]string, paginas []string) (string, error) {
	toks, err := tokenizaTemplate(tpl)
	if err != nil {
		return "", err
	}
	var antes, bloco, depois []tokenTpl
	estado := 0 // 0 antes, 1 dentro, 2 depois
	for _, t := range toks {
		switch {
		case t.tipo == "inicio":
			if estado != 0 {
				return "", fmt.Errorf("modelo, linha %d: {{#Pages}} repetido ou aninhado", t.linha)
			}
			estado = 1
		case t.tipo == "fim":
			if estado != 1 {
				return "", fmt.Errorf("modelo, linha %d: {{/Pages}} sem {{#Pages}}", t.linha)
			}
			estado = 2
		case estado == 0:
			antes = append(antes, t)
		case estado == 1:
			bloco = append(bloco, t)
		default:
			depois = append(depois, t)
		}
	}
	if estado == 1 {
		return "", fmt.Errorf("modelo: {{#Pages}} sem {{/Pages}}")
	}
	if estado == 0 {
		return "", fmt.Errorf("modelo: falta o bloco {{#Pages}} ... {{/Pages}}")
	}
	var sb strings.Builder
	emite := func(ts []tokenTpl, pagina int, texto string, dentro bool) error {
		for _, t := range ts {
			if t.tipo == "texto" {
				sb.WriteString(t.valor)
				continue
			}
			switch {
			case ehEmLista(variaveisDePagina, t.valor):
				if !dentro {
					return fmt.Errorf("modelo, linha %d: {{%s}} só vale dentro de {{#Pages}}", t.linha, t.valor)
				}
				if t.valor == "PageNumber" {
					sb.WriteString(strconv.Itoa(pagina))
				} else {
					sb.WriteString(html.EscapeString(texto))
				}
			default:
				v, ok := globais[t.valor]
				if !ok {
					return fmt.Errorf("modelo, linha %d: variável desconhecida {{%s}} (use --var %s=valor)", t.linha, t.valor, t.valor)
				}
				sb.WriteString(html.EscapeString(v))
			}
		}
		return nil
	}
	if err := emite(antes, 0, "", false); err != nil {
		return "", err
	}
	for i, p := range paginas {
		if err := emite(bloco, i+1, p, true); err != nil {
			return "", err
		}
	}
	if err := emite(depois, 0, "", false); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// logoDataURI lê a imagem e a devolve como data URI, para o HTML ser um arquivo só.
func logoDataURI(caminho string) (string, error) {
	tipos := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
	}
	tipo, ok := tipos[strings.ToLower(filepath.Ext(caminho))]
	if !ok {
		return "", fmt.Errorf("--logo: extensão não suportada em %s (use png, jpg, gif, svg ou webp)", caminho)
	}
	b, err := os.ReadFile(caminho)
	if err != nil {
		return "", err
	}
	return "data:" + tipo + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// colunasMax devolve a maior largura (em runes) entre as linhas de todas as páginas.
func colunasMax(paginas [][]string) int {
	n := 1
	for _, p := range paginas {
		for _, l := range p {
			if w := utf8.RuneCountInString(l); w > n {
				n = w
			}
		}
	}
	return n
}

// modeloPadrao devolve o modelo embutido (folhas A4); com logo, o cabeçalho ganha a imagem.
func modeloPadrao(comLogo bool) string {
	logo := ""
	if comLogo {
		logo = `<img class="logo" src="{{Logo}}" alt="">`
	}
	return strings.Replace(modeloPadraoHTML, "<!--LOGO-->", logo, 1)
}

const modeloPadraoHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{Title}}</title>
<style>
  @page { size: A4; margin: 0; }
  * { box-sizing: border-box; }
  body { margin: 0; background: #e8e8e8; font-family: "DejaVu Sans Mono", "Liberation Mono", "Courier New", monospace; }
  .sheet { width: 210mm; height: 297mm; margin: 8mm auto; padding: 14mm 15mm; background: #fff;
           box-shadow: 0 1px 6px rgba(0,0,0,.25); display: flex; flex-direction: column; overflow: hidden;
           page-break-after: always; break-after: page; }
  .sheet > header { display: flex; align-items: center; gap: 6mm; border-bottom: .3mm solid #888;
                    padding-bottom: 3mm; margin-bottom: 4mm; font-family: sans-serif; }
  .sheet > header h1 { margin: 0; font-size: 14pt; font-weight: 600; }
  .logo { max-height: 16mm; max-width: 50mm; }
  pre { margin: 0; flex: 1; line-height: 1.12; white-space: pre; overflow: hidden;
        font-size: min(11pt, calc(180mm / ({{Columns}} * 0.6))); }
  .sheet > footer { display: flex; justify-content: space-between; border-top: .3mm solid #888;
                    padding-top: 2mm; margin-top: 4mm; font: 8pt sans-serif; color: #444; }
  @media print { body { background: none; } .sheet { margin: 0; box-shadow: none; } }
</style>
</head>
<body>
{{#Pages}}
<section class="sheet">
  <header><!--LOGO--><h1>{{Title}}</h1></header>
  <pre>{{Text}}</pre>
  <footer><span>{{Date}} {{Time}}</span><span>Page {{PageNumber}} of {{PageCount}}</span></footer>
</section>
{{/Pages}}
</body>
</html>
`

// montaHTML junta as páginas do relatório ao modelo (--template ou o modelo padrão).
func montaHTML(op opcoes, paginas [][]string) (string, error) {
	tpl := ""
	if op.template != "" {
		b, err := os.ReadFile(op.template)
		if err != nil {
			return "", err
		}
		tpl = string(b)
	} else {
		tpl = modeloPadrao(op.logo != "")
	}
	// com -e não há arquivo de script: {{Script}} fica vazio e o título padrão é "rptgo"
	arquivo, titulo := "", "rptgo"
	if !op.temTextoScript {
		arquivo = filepath.Base(op.script)
		titulo = strings.TrimSuffix(arquivo, filepath.Ext(arquivo))
	}
	if op.titulo != "" {
		titulo = op.titulo
	}
	agora := report.Agora()
	globais := map[string]string{
		"Title":     titulo,
		"Script":    arquivo,
		"Date":      agora.Format("2006-01-02"),
		"Time":      agora.Format("15:04"),
		"Columns":   strconv.Itoa(colunasMax(paginas)),
		"PageCount": strconv.Itoa(len(paginas)),
		"Logo":      "",
	}
	if op.logo != "" {
		uri, err := logoDataURI(op.logo)
		if err != nil {
			return "", err
		}
		globais["Logo"] = uri
	}
	for _, kv := range op.vars {
		k, v, _ := strings.Cut(kv, "=")
		if ehEmLista(variaveisGlobais, k) || ehEmLista(variaveisDePagina, k) {
			return "", fmt.Errorf("--var %s: nome reservado (variáveis embutidas: %s, %s)", k,
				strings.Join(variaveisGlobais, ", "), strings.Join(variaveisDePagina, ", "))
		}
		globais[k] = v
	}
	textos := make([]string, len(paginas))
	for i, p := range paginas {
		textos[i] = strings.Join(p, "\n")
	}
	return renderizaTemplate(tpl, globais, textos)
}
