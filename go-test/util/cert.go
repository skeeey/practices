package util

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

const RSAPrivateKeyBlockType = "RSA PRIVATE KEY"

type CertPairs struct {
	CA                  []byte
	CAKey               []byte
	ServerCert          []byte
	ServerKey           []byte
	ShortTimeClientCert []byte
	LongTimeClientCert  []byte
	ClientKey           []byte
}

func NewCertPairs() (*CertPairs, error) {
	caKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	caCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "open-cluster-management.io"}, caKey)
	if err != nil {
		return nil, err
	}

	serverKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serverCertDERBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test-server",
			},
			SerialNumber: big.NewInt(1),
			NotBefore:    caCert.NotBefore,
			NotAfter:     caCert.NotAfter,
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			DNSNames:     []string{"localhost"},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		},
		caCert,
		serverKey.Public(),
		caKey,
	)
	if err != nil {
		return nil, err
	}

	serverCert, err := x509.ParseCertificate(serverCertDERBytes)
	if err != nil {
		return nil, err
	}

	clientKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	clientShortCertDERBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test-client",
			},
			SerialNumber: big.NewInt(1),
			NotBefore:    caCert.NotBefore,
			NotAfter:     caCert.NotBefore.Add(5 * time.Second).UTC(),
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		caCert,
		clientKey.Public(),
		caKey,
	)
	if err != nil {
		return nil, err
	}

	clientLongCertDERBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test-client",
			},
			SerialNumber: big.NewInt(1),
			NotBefore:    caCert.NotBefore,
			NotAfter:     caCert.NotBefore.Add(20 * time.Second).UTC(),
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		caCert,
		clientKey.Public(),
		caKey,
	)
	if err != nil {
		return nil, err
	}

	clientShortCert, err := x509.ParseCertificate(clientShortCertDERBytes)
	if err != nil {
		return nil, err
	}

	clientLongCert, err := x509.ParseCertificate(clientLongCertDERBytes)
	if err != nil {
		return nil, err
	}

	return &CertPairs{
		CA: pem.EncodeToMemory(&pem.Block{
			Type:  certutil.CertificateBlockType,
			Bytes: caCert.Raw,
		}),
		CAKey: pem.EncodeToMemory(&pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(caKey),
		}),
		ServerCert: pem.EncodeToMemory(&pem.Block{
			Type:  certutil.CertificateBlockType,
			Bytes: serverCert.Raw,
		}),
		ServerKey: pem.EncodeToMemory(&pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
		}),
		ShortTimeClientCert: pem.EncodeToMemory(&pem.Block{
			Type:  certutil.CertificateBlockType,
			Bytes: clientShortCert.Raw,
		}),
		LongTimeClientCert: pem.EncodeToMemory(&pem.Block{
			Type:  certutil.CertificateBlockType,
			Bytes: clientLongCert.Raw,
		}),
		ClientKey: pem.EncodeToMemory(&pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
		}),
	}, nil
}

func AppendToCertPool(caPEM []byte) (*x509.CertPool, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if ok := certPool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("invalid CA")
	}

	return certPool, nil
}

type certificateCacheEntry struct {
	cert  *tls.Certificate
	err   error
	birth time.Time
}

// isStale returns true when this cache entry is too old to be usable
func (c *certificateCacheEntry) isStale() bool {
	return time.Since(c.birth) > time.Second
}

func newCertificateCacheEntry(certFile, keyFile string) certificateCacheEntry {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	return certificateCacheEntry{cert: &cert, err: err, birth: time.Now()}
}

// CachingCertificateLoader ensures that we don't hammer the filesystem when opening many connections
// the underlying cert files are read at most once every second
func CachingCertificateLoader(certFile, keyFile string) func() (*tls.Certificate, error) {
	current := newCertificateCacheEntry(certFile, keyFile)
	var currentMtx sync.RWMutex

	return func() (*tls.Certificate, error) {
		currentMtx.RLock()
		if current.isStale() {
			currentMtx.RUnlock()

			currentMtx.Lock()
			defer currentMtx.Unlock()

			if current.isStale() {
				current = newCertificateCacheEntry(certFile, keyFile)
			}
		} else {
			defer currentMtx.RUnlock()
		}

		return current.cert, current.err
	}
}
