package sunat

import (
	"math/big"
	"strings"

	"github.com/beevik/etree"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/jurisdictions/pe"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
)

const (
	nsInvoice   = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	nsCAC       = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	nsCBC       = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	nsDS        = "http://www.w3.org/2000/09/xmldsig#"
	nsEXT       = "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
	signatureID = "SignFUDIA"
	// Catalog 51: 0101 = venta interna. Catalog 07: 10 = gravado, operación onerosa.
	operationType  = "0101"
	igvAffectation = "10"
)

// rootNamespaces are declared on the root and inherited by the signature;
// the signer needs them to canonicalize SignedInfo in place.
var rootNamespaces = [][2]string{{"xmlns", nsInvoice}, {"xmlns:cac", nsCAC}, {"xmlns:cbc", nsCBC}, {"xmlns:ds", nsDS}, {"xmlns:ext", nsEXT}}

func child(parent *etree.Element, tag string) *etree.Element { return parent.CreateElement(tag) }
func text(parent *etree.Element, tag, value string) *etree.Element {
	el := parent.CreateElement(tag)
	el.SetText(value)
	return el
}
func amount(parent *etree.Element, tag, currency, value string) {
	text(parent, tag, value).CreateAttr("currencyID", currency)
}

// percent renders a fraction as a percentage ("0.105000" → "10.5").
func percent(rate string) string {
	r, ok := new(big.Rat).SetString(rate)
	if !ok {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(new(big.Rat).Mul(r, big.NewRat(100, 1)).FloatString(4), "0"), ".")
}

// unitValue is the price without tax per unit, up to 10 decimals.
func unitValue(net, quantity string) string {
	n, okN := new(big.Rat).SetString(net)
	q, okQ := new(big.Rat).SetString(quantity)
	if !okN || !okQ || q.Sign() == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(new(big.Rat).Quo(n, q).FloatString(10), "0"), ".")
}

func quantityText(value string) string {
	return strings.TrimRight(strings.TrimRight(value, "0"), ".")
}

func ifEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// buildInvoice renders a factura (01) or boleta (03) in UBL 2.1 as SUNAT's
// guide describes. IGV and IPM share tax scheme 1000 (IPM is part of IGV), so
// the total rate of the components goes in a single IGV subtotal.
func buildInvoice(d ports.Document) *etree.Document {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	root := doc.CreateElement("Invoice")
	for _, ns := range rootNamespaces {
		root.CreateAttr(ns[0], ns[1])
	}
	child(child(child(root, "ext:UBLExtensions"), "ext:UBLExtension"), "ext:ExtensionContent")
	text(root, "cbc:UBLVersionID", "2.1")
	text(root, "cbc:CustomizationID", "2.0")
	text(root, "cbc:ID", d.FormattedNumber)
	text(root, "cbc:IssueDate", d.IssueDate)
	text(root, "cbc:InvoiceTypeCode", d.FiscalCode).CreateAttr("listID", operationType)
	text(root, "cbc:Note", pe.AmountInWords(d.Breakdown.Total, d.Currency)).CreateAttr("languageLocaleID", "1000")
	text(root, "cbc:DocumentCurrencyCode", d.Currency)

	signature := child(root, "cac:Signature")
	text(signature, "cbc:ID", d.Issuer.TaxID)
	party := child(signature, "cac:SignatoryParty")
	text(child(party, "cac:PartyIdentification"), "cbc:ID", d.Issuer.TaxID)
	text(child(party, "cac:PartyName"), "cbc:Name", d.Issuer.LegalName)
	text(child(child(signature, "cac:DigitalSignatureAttachment"), "cac:ExternalReference"), "cbc:URI", "#"+signatureID)

	supplier := child(child(root, "cac:AccountingSupplierParty"), "cac:Party")
	text(child(supplier, "cac:PartyIdentification"), "cbc:ID", d.Issuer.TaxID).CreateAttr("schemeID", "6")
	text(child(supplier, "cac:PartyName"), "cbc:Name", ifEmpty(d.Issuer.TradeName, d.Issuer.LegalName))
	legal := child(supplier, "cac:PartyLegalEntity")
	text(legal, "cbc:RegistrationName", d.Issuer.LegalName)
	address := child(legal, "cac:RegistrationAddress")
	// 0000 = domicilio fiscal (código de establecimiento anexo).
	text(address, "cbc:AddressTypeCode", "0000")
	if strings.TrimSpace(d.Issuer.Address) != "" {
		text(child(address, "cac:AddressLine"), "cbc:Line", d.Issuer.Address)
	}
	text(child(address, "cac:Country"), "cbc:IdentificationCode", "PE")

	customer := child(child(root, "cac:AccountingCustomerParty"), "cac:Party")
	text(child(customer, "cac:PartyIdentification"), "cbc:ID", ifEmpty(d.Customer.DocumentNumber, "-")).CreateAttr("schemeID", d.CustomerFiscalCode)
	customerLegal := child(customer, "cac:PartyLegalEntity")
	text(customerLegal, "cbc:RegistrationName", ifEmpty(d.Customer.Name, "CLIENTES VARIOS"))
	if strings.TrimSpace(d.Customer.Address) != "" {
		text(child(child(customerLegal, "cac:RegistrationAddress"), "cac:AddressLine"), "cbc:Line", d.Customer.Address)
	}

	// Facturas declare the payment form (catalog: Contado).
	if d.FiscalCode == "01" {
		terms := child(root, "cac:PaymentTerms")
		text(terms, "cbc:ID", "FormaPago")
		text(terms, "cbc:PaymentMeansID", "Contado")
	}

	rate := totalRate(d)
	taxTotal := child(root, "cac:TaxTotal")
	amount(taxTotal, "cbc:TaxAmount", d.Currency, d.Breakdown.TaxTotal)
	taxSubtotal(taxTotal, d.Currency, d.Breakdown.NetTotal, d.Breakdown.TaxTotal, "")

	totals := child(root, "cac:LegalMonetaryTotal")
	amount(totals, "cbc:LineExtensionAmount", d.Currency, d.Breakdown.NetTotal)
	amount(totals, "cbc:TaxInclusiveAmount", d.Currency, d.Breakdown.Total)
	amount(totals, "cbc:PayableAmount", d.Currency, d.Breakdown.Total)

	for _, l := range d.Breakdown.Lines {
		line := child(root, "cac:InvoiceLine")
		text(line, "cbc:ID", itoa(l.No))
		text(line, "cbc:InvoicedQuantity", quantityText(l.Quantity)).CreateAttr("unitCode", ifEmpty(l.UnitCode, "NIU"))
		amount(line, "cbc:LineExtensionAmount", d.Currency, l.NetAmount)
		alt := child(child(line, "cac:PricingReference"), "cac:AlternativeConditionPrice")
		amount(alt, "cbc:PriceAmount", d.Currency, l.UnitPrice)
		text(alt, "cbc:PriceTypeCode", "01")
		lineTax := child(line, "cac:TaxTotal")
		amount(lineTax, "cbc:TaxAmount", d.Currency, l.TaxAmount)
		taxSubtotal(lineTax, d.Currency, l.NetAmount, l.TaxAmount, rate)
		text(child(line, "cac:Item"), "cbc:Description", l.Description)
		amount(child(line, "cac:Price"), "cbc:PriceAmount", d.Currency, unitValue(l.NetAmount, l.Quantity))
	}
	return doc
}

func taxSubtotal(parent *etree.Element, currency, taxable, tax, rate string) {
	subtotal := child(parent, "cac:TaxSubtotal")
	amount(subtotal, "cbc:TaxableAmount", currency, taxable)
	amount(subtotal, "cbc:TaxAmount", currency, tax)
	category := child(subtotal, "cac:TaxCategory")
	if rate != "" {
		text(category, "cbc:Percent", rate)
		text(category, "cbc:TaxExemptionReasonCode", igvAffectation)
	}
	scheme := child(category, "cac:TaxScheme")
	text(scheme, "cbc:ID", "1000")
	text(scheme, "cbc:Name", "IGV")
	text(scheme, "cbc:TaxTypeCode", "VAT")
}

func totalRate(d ports.Document) string {
	total := new(big.Rat)
	for _, t := range d.Breakdown.Taxes {
		if r, ok := new(big.Rat).SetString(t.Rate); ok {
			total.Add(total, r)
		}
	}
	return percent(total.FloatString(6))
}

func itoa(n int) string { return big.NewInt(int64(n)).String() }
