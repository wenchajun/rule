/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math"
	"math/big"
	"time"
)

func priKeyHash(priKey *ecdsa.PrivateKey) []byte {
	hash := sha256.New()
	hash.Write(elliptic.Marshal(priKey.Curve, priKey.PublicKey.X, priKey.PublicKey.Y))
	return hash.Sum(nil)
}

func CreateCa(cn string) ([]byte, []byte, []byte, error) {
	// Create public/private privateKey pair of root ca.
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create root ca.
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	notBefore := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(math.MaxInt64).UTC(),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Subject: pkix.Name{
			OrganizationalUnit: []string{"kubesphere"},
			CommonName:         cn,
		},
		SubjectKeyId: priKeyHash(rootKey),
	}

	rootCertEncode, err := x509.CreateCertificate(rand.Reader, &template, &template, rootKey.Public(), rootKey)
	if err != nil {
		return nil, nil, nil, err
	}
	rootCrt := &bytes.Buffer{}
	err = pem.Encode(rootCrt, &pem.Block{Type: "CERTIFICATE", Bytes: rootCertEncode})
	if err != nil {
		return nil, nil, nil, err
	}

	// Sign for cn
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	bobPriKeyEncode, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	serverKey := &bytes.Buffer{}
	err = pem.Encode(serverKey, &pem.Block{Type: "EC PRIVATE KEY", Bytes: bobPriKeyEncode})
	if err != nil {
		return nil, nil, nil, err
	}

	bobSerialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	notBefore = time.Now().Add(-5 * time.Minute).UTC()
	bobTemplate := x509.Certificate{
		SerialNumber:          bobSerialNumber,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(math.MaxInt64).UTC(),
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Subject: pkix.Name{
			Organization: []string{"kubesphere"},
			CommonName:   cn,
		},
		SubjectKeyId: priKeyHash(privateKey),
	}

	parent, err := x509.ParseCertificate(rootCertEncode)
	if err != nil {
		return nil, nil, nil, err
	}

	certEncode, err := x509.CreateCertificate(rand.Reader, &bobTemplate, parent, privateKey.Public(), rootKey)
	if err != nil {
		return nil, nil, nil, err
	}

	serverCrt := &bytes.Buffer{}
	err = pem.Encode(serverCrt, &pem.Block{Type: "CERTIFICATE", Bytes: certEncode})
	if err != nil {
		return nil, nil, nil, err
	}

	//return Base64Encode(rootCrt.Bytes()), Base64Encode(serverKey.Bytes()), Base64Encode(serverCrt.Bytes()), nil
	return rootCrt.Bytes(), serverKey.Bytes(), serverCrt.Bytes(), nil
}

func Base64Encode(src []byte) []byte {
	enc := base64.StdEncoding
	dst := make([]byte, enc.EncodedLen(len(src)))

	enc.Encode(dst, src)
	return dst
}

func Base64Decode(src []byte) ([]byte, error) {
	enc := base64.StdEncoding
	dst := make([]byte, enc.DecodedLen(len(src)))

	_, err := enc.Decode(dst, src)
	return dst, err
}
