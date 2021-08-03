package saml

import (
	"crypto/rsa"
	"encoding/json"
	"testing"
)

const ValidPublicCert = `-----BEGIN CERTIFICATE-----
MIIEXTCCAsWgAwIBAgIJAM6I9eCQdTglMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTkwODEzMTQ1MDE1WhcNMjkwODEyMTQ1MDE1WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIB
igKCAYEA2Bf0H/hW/P8FbuTnuSVGTX+12ORmW3jhMokTefEd/cysfxXycuIaQ3oK
4WFoxlHhwkkdrmabB4b/LsJyn9g40tVi9TRKnD7RzR3gCvIAEb7ldX2B78/0VtQn
zcUAt91qo84AOHk7kY5R8fRhtP23n49F0Wk5QGczcVf9fC7td2hKAKbXQskIeylI
ElwNC8j9q/QIE1pHlc1/vHZmKuE6pPqazto8sJkcVXD7Yqel7kmhYFrR9GUnbS6/
HHB4oQ4MInI+kHgBmYM3ctVe2Dsvdj4eGEbxiYYY11ynj5jofuiib0FrZWUNwoYD
JSLEavl5Rwsn5i2pBxmGXHNez5se8qrAPQnKGBsUYn13102CnIwTKPlMUlYq0JKU
Kd20bxtgTqOgHffSL2BGEj8ojqBIUU/ewkjz+fD+yVujDjp/Lx9jZ0WKuUYZiuTU
AeWFKyp9UQdl4xhzFFMkeseZHg4wQGPqFTc8KKZ27IWmygF4J5SVSDv7hbd90bVP
wvVEvzuRAgMBAAGjUDBOMB0GA1UdDgQWBBQQliE+bg7SY3M68U03oo5YZohhvjAf
BgNVHSMEGDAWgBQQliE+bg7SY3M68U03oo5YZohhvjAMBgNVHRMEBTADAQH/MA0G
CSqGSIb3DQEBCwUAA4IBgQCIXJBtUo6NZWIqVXMcgSN/79VnrtdNR53FMyehO/Bn
S/OCD78V6nhsdIXlXQFvamgbTb0+HLIjicrta3rIwl03pIAzD8kKkeYntkD7hhnB
I30CDxeDhTOWo+pi8JlPLl9KIY6kk5Yt777CZzLe2bhTKBZiL+ybKbbppFZmpLj9
QeIRsgyb63ufq1XGVjeXtlHjeE1KJUva367oTNJ2wasgbumCAOAHmQ/dweO+WxeN
rjSAMyc1MFtHnuR+8XLiSh3xjA2mG0oMYxAroOpWVqHmrHfsCBvDoMoBo2AkyezF
pUfaD83aE5UMDjOTOFbOXdQec8HG2kPjqjhP27nL+oyWfstG32xtv7Q1nxD+iJ+H
0qeiX3/RTnJ+l878FpEK8LjuzYBcctqj8Ioqu9oUE2U2xMDQeXzG55v9l6UyT1Hu
yfJxr9o/f6YzQyuyuf7gO/X57PEF/t/EByTFDlnZLzq9nE45xPHX7mv/ASczw1QT
UVj3mPQU2/GgAW62CgKpXZE=
-----END CERTIFICATE-----`

const ValidPrivateKey = `-----BEGIN PRIVATE KEY-----
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
XZ9jcIL+Gwpfi/QLvhJrmMGJ
-----END PRIVATE KEY-----`

func TestNew(t *testing.T) {

	config := Config{
		IDPEntityID:                 "",
		SPEntityID:                  "",
		SingleSignOnURL:             "",
		SingleLogoutURL:             "",
		AudienceURI:                 "",
		AssertionConsumerServiceURL: "",
		IDPPublicCert:               ValidPublicCert,
		SPPublicCert:                ValidPublicCert,
		SPPrivateKey:                ValidPrivateKey,
		SignRequest:                 false,
		CheckResponseSigning:        false,
		RequireEncryptedAssertion:   false,
		AttributeMap:                nil,
	}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		t.Errorf("unable ot marshal config to json: %s", err)
	}

	sp, err := New(jsonConfig)
	if err != nil {
		t.Errorf("error getting new saml provider: %s", err)
	}

	if config.IDPEntityID != sp.SamlProvider.IdentityProviderIssuer {
		t.Errorf("idp entity id does not match config, want %s, got %s", config.IDPEntityID, sp.SamlProvider.IdentityProviderIssuer)
	}

	// check if sp certs were parsed properly
	spKey, err := sp.SamlProvider.GetSigningCertBytes()
	if err != nil {
		t.Errorf("uanble to get signing cert bytes: %s", err)
	}

	if string(spKey) != config.SPPublicCert {
		t.Errorf("sp signing key does not match config, want %s, got %s", config.SPPrivateKey, string(spKey))
	}
}

func Test_getCertStore(t *testing.T) {
	type args struct {
		cert string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "expect error",
			args:    args{cert: ""},
			wantErr: true,
		},
		{
			name:    "expect error 2",
			args:    args{cert: "asdf1234 not a valid cert"},
			wantErr: true,
		},
		{
			name:    "no error",
			args:    args{cert: ValidPublicCert},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getCertStore(tt.args.cert)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCertStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_getRsaPrivateKey(t *testing.T) {
	type args struct {
		privateKey string
		publicCert string
	}
	tests := []struct {
		name    string
		args    args
		want    *rsa.PrivateKey
		wantErr bool
	}{
		{
			name: "error for public key",
			args: args{
				privateKey: ValidPrivateKey,
				publicCert: "",
			},
			wantErr: true,
		},
		{
			name: "error for private key",
			args: args{
				privateKey: "",
				publicCert: ValidPublicCert,
			},
			wantErr: true,
		},
		{
			name: "no error",
			args: args{
				privateKey: ValidPrivateKey,
				publicCert: ValidPublicCert,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getRsaPrivateKey(tt.args.privateKey, tt.args.publicCert)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRsaPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_pemToBase64(t *testing.T) {
	tests := []struct {
		name    string
		pemStr  string
		want    string
		wantErr bool
	}{
		{
			name:    "error expected",
			pemStr:  "",
			wantErr: true,
			want:    "",
		},
		{
			name:    "valid cert",
			pemStr:  ValidPublicCert,
			wantErr: false,
			want:    "MIIEXTCCAsWgAwIBAgIJAM6I9eCQdTglMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTkwODEzMTQ1MDE1WhcNMjkwODEyMTQ1MDE1WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEA2Bf0H/hW/P8FbuTnuSVGTX+12ORmW3jhMokTefEd/cysfxXycuIaQ3oK4WFoxlHhwkkdrmabB4b/LsJyn9g40tVi9TRKnD7RzR3gCvIAEb7ldX2B78/0VtQnzcUAt91qo84AOHk7kY5R8fRhtP23n49F0Wk5QGczcVf9fC7td2hKAKbXQskIeylIElwNC8j9q/QIE1pHlc1/vHZmKuE6pPqazto8sJkcVXD7Yqel7kmhYFrR9GUnbS6/HHB4oQ4MInI+kHgBmYM3ctVe2Dsvdj4eGEbxiYYY11ynj5jofuiib0FrZWUNwoYDJSLEavl5Rwsn5i2pBxmGXHNez5se8qrAPQnKGBsUYn13102CnIwTKPlMUlYq0JKUKd20bxtgTqOgHffSL2BGEj8ojqBIUU/ewkjz+fD+yVujDjp/Lx9jZ0WKuUYZiuTUAeWFKyp9UQdl4xhzFFMkeseZHg4wQGPqFTc8KKZ27IWmygF4J5SVSDv7hbd90bVPwvVEvzuRAgMBAAGjUDBOMB0GA1UdDgQWBBQQliE+bg7SY3M68U03oo5YZohhvjAfBgNVHSMEGDAWgBQQliE+bg7SY3M68U03oo5YZohhvjAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBgQCIXJBtUo6NZWIqVXMcgSN/79VnrtdNR53FMyehO/BnS/OCD78V6nhsdIXlXQFvamgbTb0+HLIjicrta3rIwl03pIAzD8kKkeYntkD7hhnBI30CDxeDhTOWo+pi8JlPLl9KIY6kk5Yt777CZzLe2bhTKBZiL+ybKbbppFZmpLj9QeIRsgyb63ufq1XGVjeXtlHjeE1KJUva367oTNJ2wasgbumCAOAHmQ/dweO+WxeNrjSAMyc1MFtHnuR+8XLiSh3xjA2mG0oMYxAroOpWVqHmrHfsCBvDoMoBo2AkyezFpUfaD83aE5UMDjOTOFbOXdQec8HG2kPjqjhP27nL+oyWfstG32xtv7Q1nxD+iJ+H0qeiX3/RTnJ+l878FpEK8LjuzYBcctqj8Ioqu9oUE2U2xMDQeXzG55v9l6UyT1HuyfJxr9o/f6YzQyuyuf7gO/X57PEF/t/EByTFDlnZLzq9nE45xPHX7mv/ASczw1QTUVj3mPQU2/GgAW62CgKpXZE=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pemToBase64(tt.pemStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("pemToBase64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("pemToBase64() got = %v, want %v", got, tt.want)
			}
		})
	}
}
