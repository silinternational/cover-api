package saml

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/gobuffalo/buffalo"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/silinternational/cover-api/auth"
	"github.com/silinternational/cover-api/domain"
)

const (
	keyTypeCert    = "CERTIFICATE"
	keyTypePrivate = "PRIVATE KEY"
)

type Provider struct {
	Config       Config
	SamlProvider *saml2.SAMLServiceProvider
}

type Config struct {
	IDPEntityID                 string            `json:"IDPEntityID"`
	SPEntityID                  string            `json:"SPEntityID"`
	SingleSignOnURL             string            `json:"SingleSignOnURL"`
	SingleLogoutURL             string            `json:"SingleLogoutURL"`
	AudienceURI                 string            `json:"AudienceURI"`
	AssertionConsumerServiceURL string            `json:"AssertionConsumerServiceURL"`
	IDPPublicCert               string            `json:"IDPPublicCert"`
	IDPPublicCert2              string            `json:"IDPPublicCert2"`
	SPPublicCert                string            `json:"SPPublicCert"`
	SPPrivateKey                string            `json:"SPPrivateKey"`
	SignRequest                 bool              `json:"SignRequest"`
	CheckResponseSigning        bool              `json:"CheckResponseSigning"`
	RequireEncryptedAssertion   bool              `json:"RequireEncryptedAssertion"`
	AttributeMap                map[string]string `json:"AttributeMap"`
}

// GetKeyPair implements dsig.X509KeyStore interface
func (c *Config) GetKeyPair() (privateKey *rsa.PrivateKey, cert []byte, err error) {
	rsaKey, err := getRsaPrivateKey(c.SPPrivateKey, c.SPPublicCert)
	if err != nil {
		return &rsa.PrivateKey{}, []byte{}, err
	}

	return rsaKey, []byte(c.SPPublicCert), nil
}

func New(config Config) (*Provider, error) {
	p := &Provider{
		Config: config,
	}

	err := p.initSAMLServiceProvider()
	if err != nil {
		return p, err
	}

	return p, nil
}

func (p *Provider) initSAMLServiceProvider() error {
	idpCertStore, err := getCertStore(p.Config.IDPPublicCert, p.Config.IDPPublicCert2)
	if err != nil {
		return fmt.Errorf("error in initSAMLServiceProvider: %w", err)
	}

	p.SamlProvider = &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:         p.Config.SingleSignOnURL,
		IdentityProviderIssuer:         p.Config.IDPEntityID,
		AssertionConsumerServiceURL:    p.Config.AssertionConsumerServiceURL,
		ServiceProviderIssuer:          p.Config.SPEntityID,
		SignAuthnRequests:              p.Config.SignRequest,
		SignAuthnRequestsAlgorithm:     "",
		SignAuthnRequestsCanonicalizer: nil,
		RequestedAuthnContext:          nil,
		AudienceURI:                    p.Config.AudienceURI,
		IDPCertificateStore:            &idpCertStore,
		SPKeyStore:                     &p.Config,
		SPSigningKeyStore:              &p.Config,
		NameIdFormat:                   "",
		ValidateEncryptionCert:         false,
		SkipSignatureValidation:        false,
		AllowMissingAttributes:         false,
		Clock:                          nil,
	}

	return nil
}

// AuthRequest returns the URL for the authentication end-point
func (p *Provider) AuthRequest(c buffalo.Context) (string, error) {
	return p.SamlProvider.BuildAuthURL("")
}

// AuthCallback gets information about the user from the saml assertion.
func (p *Provider) AuthCallback(c buffalo.Context) auth.Response {
	resp := auth.Response{}

	// check if this is not a saml response and redirect
	samlResp := c.Param("SAMLResponse")
	if samlResp == "" {
		resp.RedirectURL, resp.Error = p.SamlProvider.BuildAuthURL("")
		return resp
	}

	// verify and retrieve assertion
	assertion, err := p.SamlProvider.RetrieveAssertionInfo(samlResp)
	if err != nil {
		resp.Error = err
		return resp
	}
	resp.AuthUser = getUserFromAssertion(assertion)

	return resp
}

func (p *Provider) Logout(c buffalo.Context) auth.Response {
	resp := auth.Response{}
	err := auth.Logout(c.Response(), c.Request())
	if err != nil {
		resp.Error = err
	}
	rURL := fmt.Sprintf("%s?ReturnTo=%s", p.Config.SingleLogoutURL, domain.LogoutRedirectURL)
	return auth.Response{RedirectURL: rURL}
}

func getUserFromAssertion(assertion *saml2.AssertionInfo) *auth.User {
	return &auth.User{
		FirstName: getSAMLAttributeFirstValue("givenName", assertion.Assertions[0].AttributeStatement.Attributes),
		LastName:  getSAMLAttributeFirstValue("sn", assertion.Assertions[0].AttributeStatement.Attributes),
		Email:     getSAMLAttributeFirstValue("mail", assertion.Assertions[0].AttributeStatement.Attributes),
		StaffID:   getSAMLAttributeFirstValue("employeeNumber", assertion.Assertions[0].AttributeStatement.Attributes),
	}
}

func getSAMLAttributeFirstValue(attrName string, attributes []types.Attribute) string {
	for _, attr := range attributes {
		if attr.Name != attrName {
			continue
		}

		if len(attr.Values) > 0 {
			return attr.Values[0].Value
		}
		return ""
	}
	return ""
}

func getCertStore(certs ...string) (dsig.MemoryX509CertificateStore, error) {
	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	if len(certs) < 1 || certs[0] == "" {
		return certStore, errors.New("a valid PEM or base64 encoded certificate is required")
	}

	for _, cert := range certs {
		if cert == "" {
			continue
		}

		certData, err := decodeKey(cert, "CERTIFICATE")
		if err != nil {
			return certStore, fmt.Errorf("error decoding cert from string %q: %s", cert, err)
		}

		idpCert, err := x509.ParseCertificate(certData)
		if err != nil {
			return certStore, fmt.Errorf("error parsing cert: %s", err)
		}

		certStore.Roots = append(certStore.Roots, idpCert)
	}

	return certStore, nil
}

func getRsaPrivateKey(privateKey, publicCert string) (*rsa.PrivateKey, error) {
	var rsaKey *rsa.PrivateKey

	if privateKey == "" {
		return rsaKey, errors.New("A valid PEM or base64 encoded privateKey is required")
	}

	if publicCert == "" {
		return rsaKey, errors.New("A valid PEM or base64 encoded publicCert is required")
	}

	privateKeyBytes, err := decodeKey(privateKey, keyTypePrivate)
	if err != nil {
		return nil, fmt.Errorf("problem with RSA private key: %w", err)
	}

	var parsedKey any
	if parsedKey, err = x509.ParsePKCS8PrivateKey(privateKeyBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS1PrivateKey(privateKeyBytes); err != nil {
			return rsaKey, fmt.Errorf("unable to parse RSA private key: %s", err)
		}
	}

	var ok bool
	rsaKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		return rsaKey, errors.New("unable to assert parsed key type")
	}

	publicCertBytes, err := decodeKey(publicCert, keyTypeCert)
	if err != nil {
		return nil, fmt.Errorf("problem with RSA public cert: %w", err)
	}

	cert, err := x509.ParseCertificate(publicCertBytes)
	if err != nil {
		return rsaKey, fmt.Errorf("unable to parse RSA public cert: %s", err)
	}

	var pubKey *rsa.PublicKey
	if pubKey, ok = cert.PublicKey.(*rsa.PublicKey); !ok {
		return rsaKey, errors.New("unable to assert RSA public cert type")
	}

	rsaKey.PublicKey = *pubKey

	return rsaKey, nil
}

func pemToBase64(pemStr string) (string, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return "", errors.New("input string is not PEM encoded")
	}

	return base64.StdEncoding.EncodeToString(block.Bytes), nil
}

// decodeKey decodes a key from either a PEM-encoded string or a base64 string
func decodeKey(key, expectedType string) ([]byte, error) {
	block, _ := pem.Decode([]byte(key))
	if block != nil {
		if block.Type != expectedType {
			return nil, fmt.Errorf("key is of the wrong type, expected %s but found %s", expectedType, block.Type)
		}
		return block.Bytes, nil
	}

	var bytes []byte
	bytes = make([]byte, base64.StdEncoding.DecodedLen(len(key)))
	n, err := base64.StdEncoding.Decode(bytes, []byte(key))
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64: %w", err)
	}
	return bytes[:n], nil
}
