package saml

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/gobuffalo/buffalo"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/silinternational/riskman-api/auth"
	"github.com/silinternational/riskman-api/domain"
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

func New(jsonConfig json.RawMessage) (*Provider, error) {
	var config Config
	err := json.Unmarshal(jsonConfig, &config)
	if err != nil {
		return &Provider{}, err
	}

	p := &Provider{
		Config: config,
	}

	err = p.initSAMLServiceProvider()
	if err != nil {
		return p, err
	}

	return p, nil
}

func (p *Provider) initSAMLServiceProvider() error {

	idpCertStore, err := getCertStore(p.Config.IDPPublicCert)
	if err != nil {
		return err
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
	rURL := fmt.Sprintf("%s?ReturnTo=%s", p.Config.SingleLogoutURL, domain.Env.UIURL)
	return auth.Response{RedirectURL: rURL}
}

func getUserFromAssertion(assertion *saml2.AssertionInfo) *auth.User {
	return &auth.User{
		FirstName: getSAMLAttributeFirstValue("givenName", assertion.Assertions[0].AttributeStatement.Attributes),
		LastName:  getSAMLAttributeFirstValue("sn", assertion.Assertions[0].AttributeStatement.Attributes),
		Email:     getSAMLAttributeFirstValue("mail", assertion.Assertions[0].AttributeStatement.Attributes),
		UserID:    assertion.NameID,
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

func getCertStore(cert string) (dsig.MemoryX509CertificateStore, error) {
	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	if cert == "" || !strings.HasPrefix(cert, "-") {
		return certStore, errors.New("a valid PEM encoded certificate is required")
	}

	cert, err := pemToBase64(cert)
	if err != nil {
		return certStore, err
	}

	certData, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return certStore, fmt.Errorf("error decoding cert from string: %s", err)
	}

	idpCert, err := x509.ParseCertificate(certData)
	if err != nil {
		return certStore, fmt.Errorf("error parsing cert: %s", err)
	}

	certStore.Roots = append(certStore.Roots, idpCert)

	return certStore, nil
}

func getRsaPrivateKey(privateKey, publicCert string) (*rsa.PrivateKey, error) {
	var rsaKey *rsa.PrivateKey

	if privateKey == "" {
		return rsaKey, errors.New("A valid PEM encoded privateKey is required")
	}

	if publicCert == "" {
		return rsaKey, errors.New("A valid PEM encoded publicCert is required")
	}

	privPem, _ := pem.Decode([]byte(privateKey))
	if privPem.Type != "PRIVATE KEY" {
		return rsaKey, fmt.Errorf("RSA private key is of the wrong type: %s", privPem.Type)
	}

	var err error
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes); err != nil {
		if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
			return rsaKey, fmt.Errorf("unable to parse RSA private key: %s", err)
		}
	}

	var ok bool
	rsaKey, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		return rsaKey, errors.New("unable to assert parsed key type")
	}

	pubPem, _ := pem.Decode([]byte(publicCert))
	if pubPem == nil {
		return rsaKey, errors.New("rsa public key not in pem format")
	}
	if pubPem.Type != "CERTIFICATE" {
		return rsaKey, fmt.Errorf("RSA public key is of the wrong type: %s", pubPem.Type)
	}

	cert, err := x509.ParseCertificate(pubPem.Bytes)
	if err != nil {
		return rsaKey, fmt.Errorf("unable to parse RSA public key: %s", err)
	}

	var pubKey *rsa.PublicKey
	if pubKey, ok = cert.PublicKey.(*rsa.PublicKey); !ok {
		return rsaKey, fmt.Errorf("unable to parse RSA public key: %s", err)
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
