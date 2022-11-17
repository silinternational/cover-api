package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// This code was informed by crypto/tls/generate_cert.go in the Go source repository

// generateCert creates a new self-signed certificate
func generateCert(certFile, keyFile string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"SIL International"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// for self-cert
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	// #nosec G304 -- ok to use a filename from a variable since this is not exported code
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("open cert file '%s': %w", certFile, err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("write cert file '%s': %w", certFile, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("close cert file '%s': %w", certFile, err)
	}

	// #nosec G304 -- ok to use a filename from a variable since this is not exported code
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open key file '%s': %w", keyFile, err)
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	if err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("write key file '%s': %w", keyFile, err)
	}
	if err = keyOut.Close(); err != nil {
		return fmt.Errorf("close key file '%s': %w", keyFile, err)
	}

	return nil
}
