package sunat

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"errors"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

// SUNAT validates XMLDSig enveloped signatures with inclusive C14N 1.0,
// RSA-SHA1 and SHA1 digests (the profile used by Greenter and SUNAT samples).
const (
	algC14N      = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	algRSASHA1   = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
	algSHA1      = "http://www.w3.org/2000/09/xmldsig#sha1"
	algEnveloped = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
)

type signer struct {
	key  *rsa.PrivateKey
	cert *x509.Certificate
}

// sign places ds:Signature inside the empty ext:ExtensionContent and returns
// the digest value (the receipt hash).
func (s signer) sign(doc *etree.Document) (string, error) {
	root := doc.Root()
	content := root.FindElement("./ext:UBLExtensions/ext:UBLExtension/ext:ExtensionContent")
	if content == nil || len(content.ChildElements()) > 0 {
		return "", errors.New("ExtensionContent vacío no encontrado")
	}
	canonicalizer := dsig.MakeC14N10RecCanonicalizer()
	// The enveloped transform removes the signature, so the digest covers the
	// document exactly as it is before inserting it.
	body, err := canonicalizer.Canonicalize(root)
	if err != nil {
		return "", err
	}
	digestSum := sha1.Sum(body)
	digest := base64.StdEncoding.EncodeToString(digestSum[:])

	signature := etree.NewElement("ds:Signature")
	signature.CreateAttr("Id", signatureID)
	signedInfo := signature.CreateElement("ds:SignedInfo")
	signedInfo.CreateElement("ds:CanonicalizationMethod").CreateAttr("Algorithm", algC14N)
	signedInfo.CreateElement("ds:SignatureMethod").CreateAttr("Algorithm", algRSASHA1)
	reference := signedInfo.CreateElement("ds:Reference")
	reference.CreateAttr("URI", "")
	reference.CreateElement("ds:Transforms").CreateElement("ds:Transform").CreateAttr("Algorithm", algEnveloped)
	reference.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", algSHA1)
	reference.CreateElement("ds:DigestValue").SetText(digest)

	// Inclusive C14N renders every in-scope namespace on the SignedInfo apex,
	// so the verifier canonicalizes it with the root declarations.
	probe := signedInfo.Copy()
	for _, ns := range rootNamespaces {
		probe.CreateAttr(ns[0], ns[1])
	}
	canonicalInfo, err := canonicalizer.Canonicalize(probe)
	if err != nil {
		return "", err
	}
	infoSum := sha1.Sum(canonicalInfo)
	value, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA1, infoSum[:])
	if err != nil {
		return "", err
	}
	signature.CreateElement("ds:SignatureValue").SetText(base64.StdEncoding.EncodeToString(value))
	signature.CreateElement("ds:KeyInfo").CreateElement("ds:X509Data").CreateElement("ds:X509Certificate").SetText(base64.StdEncoding.EncodeToString(s.cert.Raw))
	content.AddChild(signature)
	return digest, nil
}
