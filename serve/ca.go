package serve

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	mrand "math/rand"
	"net"
	"time"
)

func init() {
	ca, err := NewCA()
	if err != nil {
		panic(err)
	}
	defaultCA = &ca
}

var defaultCA *certWrapper

type certWrapper struct {
	Bytes       []byte
	Certificate *x509.Certificate
	PrivateKey  *ecdsa.PrivateKey
}

func (c *certWrapper) Config() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{c.Bytes},
			PrivateKey:  c.PrivateKey,
		}},
		NextProtos: []string{"http/1.1"},
	}
}

func (c *certWrapper) Sign(host string) (*tls.Certificate, error) {
	t := &x509.Certificate{
		SerialNumber: big.NewInt(mrand.Int63()),
		Issuer:       c.Certificate.Subject,
		Subject:      c.Certificate.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 3, 0),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	ip := net.ParseIP(host)
	if ip != nil {
		t.IPAddresses = append(t.IPAddresses, ip)
	} else {
		t.DNSNames = append(t.DNSNames, host)
		t.Subject.CommonName = host
	}
	hostCert, err := certificateAndPrivateKey(t, c.Certificate)
	if err != nil {
		return nil, err
	}
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{hostCert.Bytes, c.Bytes},
		PrivateKey:  hostCert.PrivateKey,
	}
	return tlsCert, nil
}

func NewCA() (res certWrapper, err error) {
	ca := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		SerialNumber:          big.NewInt(2022),
		Subject: pkix.Name{
			Organization: []string{"Untrusted MITM Proxy, INC"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 3, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	return certificateAndPrivateKey(ca, ca)
}

func certificateAndPrivateKey(cert, parent *x509.Certificate) (res certWrapper, err error) {
	res.Certificate = cert
	// res.children = map[string]*tls.Certificate{}
	res.PrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return
	}
	res.Bytes, err = x509.CreateCertificate(rand.Reader, cert, parent,
		&res.PrivateKey.PublicKey, res.PrivateKey)
	if err != nil {
		return
	}
	return res, nil
}
