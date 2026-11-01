package sunat

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

// fault is a SOAP fault from SUNAT; Code is the numeric SUNAT code when present.
type fault struct {
	Code    int
	Message string
}

func (f *fault) Error() string { return fmt.Sprintf("SUNAT %04d: %s", f.Code, f.Message) }

// cdr is SUNAT's ApplicationResponse (constancia de recepción).
type cdr struct {
	ResponseCode string
	Description  string
	Notes        []string
}

var faultCode = regexp.MustCompile(`(\d{4})`)

func zipFile(name string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(name)
	if err != nil {
		return nil, err
	}
	if _, err = f.Write(content); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func escape(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

// sendBill posts the zipped, signed document and returns the zipped CDR.
func sendBill(ctx context.Context, client *http.Client, endpoint, username, password, fileName string, content []byte) ([]byte, error) {
	envelope := `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ser="http://service.sunat.gob.pe" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">` +
		`<soapenv:Header><wsse:Security><wsse:UsernameToken><wsse:Username>` + escape(username) + `</wsse:Username><wsse:Password>` + escape(password) + `</wsse:Password></wsse:UsernameToken></wsse:Security></soapenv:Header>` +
		`<soapenv:Body><ser:sendBill><fileName>` + escape(fileName) + `</fileName><contentFile>` + base64.StdEncoding.EncodeToString(content) + `</contentFile></ser:sendBill></soapenv:Body></soapenv:Envelope>`
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(envelope))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "text/xml; charset=utf-8")
	request.Header.Set("SOAPAction", "urn:sendBill")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar con SUNAT: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	return parseSendBillResponse(response.StatusCode, body)
}

func parseSendBillResponse(status int, body []byte) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("SUNAT respondió HTTP %d sin XML válido: %s", status, snippet)
	}
	if f := doc.FindElement("//Fault"); f != nil {
		codeText, message := elementText(f, "faultcode"), elementText(f, "faultstring")
		code := 0
		if match := faultCode.FindString(codeText); match != "" {
			code, _ = strconv.Atoi(match)
		} else if match := faultCode.FindString(message); match != "" {
			code, _ = strconv.Atoi(match)
		}
		return nil, &fault{Code: code, Message: strings.TrimSpace(message)}
	}
	encoded := elementText(doc.Root(), "applicationResponse")
	if encoded == "" {
		return nil, fmt.Errorf("SUNAT respondió HTTP %d sin constancia de recepción", status)
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
}

func elementText(parent *etree.Element, tag string) string {
	if parent == nil {
		return ""
	}
	if el := parent.FindElement("//" + tag); el != nil {
		return el.Text()
	}
	return ""
}

// readCDR unzips and parses the ApplicationResponse.
func readCDR(zipped []byte) (cdr, error) {
	var out cdr
	reader, err := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	if err != nil {
		return out, err
	}
	for _, file := range reader.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return out, err
		}
		content, err := io.ReadAll(io.LimitReader(rc, 5<<20))
		rc.Close()
		if err != nil {
			return out, err
		}
		doc := etree.NewDocument()
		if err := doc.ReadFromBytes(content); err != nil {
			return out, err
		}
		root := doc.Root()
		out.ResponseCode = strings.TrimSpace(elementText(root, "ResponseCode"))
		out.Description = strings.TrimSpace(elementText(root, "Description"))
		for _, note := range root.SelectElements("Note") {
			out.Notes = append(out.Notes, strings.TrimSpace(note.Text()))
		}
		return out, nil
	}
	return out, errors.New("la constancia de SUNAT no contiene XML")
}
