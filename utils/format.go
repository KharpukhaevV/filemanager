package utils

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"io"
	"strings"
)

// ===================== Форматирование контента =====================

func FormatJSON(content string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(content), "", "  ")
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func FormatXML(content string) (string, error) {
	var buf bytes.Buffer
	decoder := xml.NewDecoder(strings.NewReader(content))
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ошибка декодирования XML: %v", err)
		}

		if err := encoder.EncodeToken(token); err != nil {
			return "", fmt.Errorf("ошибка кодирования XML: %v", err)
		}
	}

	if err := encoder.Flush(); err != nil {
		return "", fmt.Errorf("ошибка завершения кодировки XML: %v", err)
	}

	result := buf.String()
	result = strings.ReplaceAll(result, "\n\n\n", "\n")
	result = strings.ReplaceAll(result, "\n\n", "\n")

	return result, nil
}

func RenderMarkdown(content string) (string, error) {
	return MarkdownToANSI(content), nil
}

func HighlightSyntax(content string, filename string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, _ := lexer.Tokenise(nil, content)
	var highlighted strings.Builder
	formatter.Format(&highlighted, style, iterator)

	return highlighted.String()
}
