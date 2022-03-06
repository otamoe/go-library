package certificate

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type (
	Certificate struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"private_key"`
	}
)

// 创建 私钥
func CreatePrivateKey(typ string, bits int) (priv crypto.PrivateKey, err error) {
	reader := rand.Reader
	switch typ {
	case "ed25519":
		{
			var privateKey ed25519.PrivateKey
			if _, privateKey, err = ed25519.GenerateKey(reader); err != nil {
				return
			}
			priv = privateKey
		}
	case "ecdsa":
		{
			var privateKey *ecdsa.PrivateKey
			switch bits {
			case 224:
				privateKey, err = ecdsa.GenerateKey(elliptic.P224(), reader)
			case 256:
				privateKey, err = ecdsa.GenerateKey(elliptic.P256(), reader)
			case 384:
				privateKey, err = ecdsa.GenerateKey(elliptic.P384(), reader)
			case 521:
				privateKey, err = ecdsa.GenerateKey(elliptic.P521(), reader)
			}
			if err != nil {
				return
			}
			priv = privateKey
		}
	default:
		{
			var privateKey *rsa.PrivateKey
			if privateKey, err = rsa.GenerateKey(reader, bits); err != nil {
				return
			}
			priv = privateKey
		}
	}
	return
}

// 编码 私钥
func MarshalPrivateKey(priv crypto.PrivateKey) (privBytes []byte, err error) {
	var blockBytes []byte
	var blockType string
	switch val := priv.(type) {
	case ed25519.PrivateKey:
		{
			if blockBytes, err = x509.MarshalPKCS8PrivateKey(val); err != nil {
				return
			}
			blockType = "Ed25519"
		}
	case *ecdsa.PrivateKey:
		{
			if blockBytes, err = x509.MarshalECPrivateKey(val); err != nil {
				return
			}
			blockType = "EC"
		}
	case *rsa.PrivateKey:
		{
			if blockBytes, err = x509.MarshalPKCS8PrivateKey(val); err != nil {
				return
			}
			blockType = "RSA"
		}
	default:
		err = errors.New("Unknown private key type")
		return
	}

	privBytes = pem.EncodeToMemory(&pem.Block{
		Type:  blockType + " PRIVATE KEY",
		Bytes: blockBytes,
	})
	return
}

// 解码 私钥匙
func UnmarshalPrivateKey(privBytes []byte) (priv crypto.PrivateKey, err error) {
	keyPEMBlock := privBytes
	var keyDERBlock *pem.Block
	for {
		keyDERBlock, keyPEMBlock = pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			break
		}
		if keyDERBlock.Type == "PRIVATE KEY" || strings.HasSuffix(keyDERBlock.Type, " PRIVATE KEY") {
			break
		}
	}
	if keyDERBlock == nil {
		err = errors.New("Private key not found")
		return
	}

	if priv, err = parsePrivateKey(keyDERBlock.Bytes); err != nil {
		return
	}
	return
}

// 读取公钥
func ReadPubliKey(priv crypto.PrivateKey) (pub crypto.PublicKey, err error) {
	if val, ok := priv.(crypto.Signer); ok {
		pub = val.Public()
		return
	}
	err = errors.New("Unknown private key type")
	return
}

// 创建 or 签名 证书
func CreateCertificate(pub crypto.PublicKey, certificate *x509.Certificate, parentCertificate *x509.Certificate, priv crypto.PrivateKey) (cert []byte, err error) {
	if certificate.SerialNumber == nil {
		max := new(big.Int).Lsh(big.NewInt(1), 128)
		var serialNumber *big.Int
		if serialNumber, err = rand.Int(rand.Reader, max); err != nil {
			return
		}
		certificate.SerialNumber = serialNumber
	}
	if certificate.NotBefore.IsZero() {
		certificate.NotBefore = time.Now().UTC().Add(0 - (time.Hour * 24 * 2))
	}

	// 	父级
	if parentCertificate == nil {
		parentCertificate = certificate
	}

	if cert, err = x509.CreateCertificate(rand.Reader, certificate, parentCertificate, pub, priv); err != nil {
		return
	}

	cert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	return
}

func CreateTLSCertificate(typ string, bits int, commonName string, domains []string, isCA bool, parent *Certificate) (certificate *Certificate, err error) {
	var priv crypto.PrivateKey
	if priv, err = CreatePrivateKey(typ, bits); err != nil {
		return
	}
	var privBytes []byte
	if privBytes, err = MarshalPrivateKey(priv); err != nil {
		return
	}
	var pub crypto.PublicKey
	if pub, err = ReadPubliKey(priv); err != nil {
		return
	}

	x509Certificate := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: commonName,
		},

		NotAfter:              time.Date(2037, time.January, 1, 0, 0, 0, 0, time.UTC),                     // 过期数据
		BasicConstraintsValid: true,                                                                       //基本的有效性约束
		IsCA:                  isCA,                                                                       // 是否是CA
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}, // 扩展证书用途 客户端认证, 服务端认证
		DNSNames:              domains,
		PermittedDNSDomains:   domains,
		PermittedURIDomains:   domains,
	}
	if typ == "ed25519" {
		x509Certificate.SignatureAlgorithm = x509.PureEd25519
	} else if typ == "ecdsa" {
		x509Certificate.SignatureAlgorithm = x509.ECDSAWithSHA256
	} else {
		x509Certificate.SignatureAlgorithm = x509.SHA256WithRSA
	}
	if isCA {
		x509Certificate.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign // 证书用途   数字签名, 证书签名, CRL签名
		x509Certificate.MaxPathLen = 4
		x509Certificate.MaxPathLenZero = false
		x509Certificate.DNSNames = nil
		x509Certificate.PermittedDNSDomains = nil
		x509Certificate.PermittedURIDomains = nil
	} else {
		x509Certificate.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment // 证书用途  数字签名, 密钥加密
	}

	var parentCertificate *x509.Certificate
	var parentPriv crypto.PrivateKey
	if parent == nil {
		parentCertificate = x509Certificate
		parentPriv = priv
	} else {
		if parentCertificate, err = UnmarshalCertificate([]byte(parent.Certificate)); err != nil {
			return
		}
		if parentPriv, err = UnmarshalPrivateKey([]byte(parent.PrivateKey)); err != nil {
			return
		}
	}

	var certificateByte []byte
	if certificateByte, err = CreateCertificate(pub, x509Certificate, parentCertificate, parentPriv); err != nil {
		return
	}

	certificate = &Certificate{PrivateKey: string(privBytes), Certificate: string(certificateByte)}
	return
}

// 解析证书
func UnmarshalCertificate(pubBytes []byte) (cert *x509.Certificate, err error) {
	pubPEMBlock := pubBytes
	var pubDERBlock *pem.Block
	for {
		pubDERBlock, pubPEMBlock = pem.Decode(pubPEMBlock)
		if pubDERBlock == nil {
			break
		}
		if pubDERBlock.Type == "CERTIFICATE" {
			break
		}
	}
	if pubDERBlock == nil {
		err = errors.New("Certificate not found")
		return
	}

	if cert, err = x509.ParseCertificate(pubDERBlock.Bytes); err != nil {
		return
	}
	return
}

func TLSConfig(certificates []*Certificate) (config *tls.Config, err error) {
	var tlsCertificates []tls.Certificate
	for _, val := range certificates {
		var certificate tls.Certificate
		if certificate, err = tls.X509KeyPair([]byte(val.Certificate), []byte(val.PrivateKey)); err != nil {
			return
		}
		tlsCertificates = append(tlsCertificates, certificate)
	}
	config = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: tlsCertificates,
	}
	return
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS #1 private keys by default, while OpenSSL 1.0.0 generates PKCS #8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("tls: failed to parse private key")
}

type pkcs1PublicKey struct {
	N *big.Int
	E int
}

func subjectKeyId(pub interface{}) (id []byte, err error) {
	var publicKeyBytes []byte
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		if publicKeyBytes, err = asn1.Marshal(pkcs1PublicKey{
			N: pub.N,
			E: pub.E,
		}); err != nil {
			return
		}
	case *ecdsa.PublicKey:
		publicKeyBytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	case ed25519.PublicKey:
		publicKeyBytes = pub
	default:
		err = fmt.Errorf("x509: unsupported public key type: %T", pub)
		return
	}

	hash := sha256.New()
	hash.Write(publicKeyBytes)
	id = hash.Sum(nil)
	return
}
