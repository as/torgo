// Copyright 2015 "as". All rights reserved. The program and its corresponding
// gotools package is governed by a BSD license.

package main

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"crypto/x509"
	"reflect"
	"strings"
)
import (
	"github.com/as/mute"
)

const (
	Prefix = "cert: "
	Debug  = false
)

var args struct {
	h, q, v bool
	k       bool
	m       bool
	a       int
	n       string
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.k, "k", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.IntVar(&args.a, "a", 4096, "")
	f.StringVar(&args.n, "n", "tcp4", "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

var (
	socket string
	proto  string
	cmd    []string
	done   chan error
)

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.k, "k", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.IntVar(&args.a, "a", 4096, "")
	f.StringVar(&args.n, "n", "tcp4", "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

var f *flag.FlagSet

type InvalidReason int
type SignatureAlgorithm int
type PublicKeyAlgorithm int
type KeyUsage x509.KeyUsage
type ExtKeyUsage x509.ExtKeyUsage

type Stringer interface {
	String() string
}

func unfuncname(t Stringer) []byte {
	name := reflect.TypeOf(t).Name()
	s := t.String()
	return []byte(`"` + strings.TrimPrefix(s, name) + `"`)
}

//func (e InvalidReason)     MarshalJSON() ([]byte, error) { return unfuncname(e), nil}
func (e SignatureAlgorithm) MarshalJSON() ([]byte, error) { return unfuncname(e), nil }
func (e PublicKeyAlgorithm) MarshalJSON() ([]byte, error) { return unfuncname(e), nil }
func (e KeyUsage) MarshalJSON() ([]byte, error)           { return unfuncname(e), nil }
func (e ExtKeyUsage) MarshalJSON() ([]byte, error)        { return unfuncname(e), nil }

type Suite uint16

func (s Suite) String() string {
	return suiteString[s]
}

const (
	RSA_RC4_128_SHA                Suite = 0x0005
	RSA_3DES_EDE_CBC_SHA           Suite = 0x000a
	RSA_AES_128_CBC_SHA            Suite = 0x002f
	RSA_AES_256_CBC_SHA            Suite = 0x0035
	ECDHE_ECDSA_RC4_128_SHA        Suite = 0xc007
	ECDHE_ECDSA_AES_128_CBC_SHA    Suite = 0xc009
	ECDHE_ECDSA_AES_256_CBC_SHA    Suite = 0xc00a
	ECDHE_RSA_RC4_128_SHA          Suite = 0xc011
	ECDHE_RSA_3DES_EDE_CBC_SHA     Suite = 0xc012
	ECDHE_RSA_AES_128_CBC_SHA      Suite = 0xc013
	ECDHE_RSA_AES_256_CBC_SHA      Suite = 0xc014
	ECDHE_RSA_AES_128_GCM_SHA256   Suite = 0xc02f
	ECDHE_ECDSA_AES_128_GCM_SHA256 Suite = 0xc02b
	ECDHE_RSA_AES_256_GCM_SHA384   Suite = 0xc030
	ECDHE_ECDSA_AES_256_GCM_SHA384 Suite = 0xc02c

	// FALLBACK_SCSV isn't a standard cipher suite but an indicator
	// that the client is doing version fallback. See
	// https://tools.ietf.org/html/draft-ietf-tls-downgrade-scsv-00.
	FALLBACK_SCSV Suite = 0x5600
)

var suiteString = map[Suite]string{
	0x0005: "RSA/RC4/128/SHA",
	0x000a: "RSA/3DES/EDE/CBC/SHA",
	0x002f: "RSA/AES/CBC/128/SHA",
	0x0035: "RSA/AES/CBC/256/SHA",
	0xc007: "ECDHE/ECDSA/RC4/128/SHA",
	0xc009: "ECDHE/ECDSA/AES/CBC/128/SHA",
	0xc00a: "ECDHE/ECDSA/AES/CBC/256/SHA",
	0xc011: "ECDHE/RSA/RC4/128/SHA",
	0xc012: "ECDHE/RSA/3DES/EDE/CBC/SHA",
	0xc013: "ECDHE/RSA/AES/CBC/128/SHA",
	0xc014: "ECDHE/RSA/AES/CBC/256/SHA",
	0xc02f: "ECDHE/RSA/AES/GCM/128/SHA256",
	0xc02b: "ECDHE/ECDSA/AES/GCM/128/SHA256",
	0xc030: "ECDHE/RSA/AES/GCM/256/SHA384",
	0xc02c: "ECDHE/ECDSA/AES/GCM/256/SHA384",
	0x5600: "FALLBACK/SCSV",
}

const (
	NotAuthorizedToSign InvalidReason = iota
	Expired
	CANotAuthorizedForThisName
	TooManyIntermediates
	IncompatibleUsage
)
const (
	UnknownSignatureAlgorithm SignatureAlgorithm = iota
	MD2WithRSA
	MD5WithRSA
	SHA1WithRSA
	SHA256WithRSA
	SHA384WithRSA
	SHA512WithRSA
	DSAWithSHA1
	DSAWithSHA256
	ECDSAWithSHA1
	ECDSAWithSHA256
	ECDSAWithSHA384
	ECDSAWithSHA512
)
const (
	UnknownPublicKeyAlgorithm PublicKeyAlgorithm = iota
	RSA
	DSA
	ECDSA
)
const (
	KeyUsageDigitalSignature KeyUsage = 1 << iota
	KeyUsageContentCommitment
	KeyUsageKeyEncipherment
	KeyUsageDataEncipherment
	KeyUsageKeyAgreement
	KeyUsageCertSign
	KeyUsageCRLSign
	KeyUsageEncipherOnly
	KeyUsageDecipherOnly
)
const (
	ExtKeyUsageAny ExtKeyUsage = iota
	ExtKeyUsageServerAuth
	ExtKeyUsageClientAuth
	ExtKeyUsageCodeSigning
	ExtKeyUsageEmailProtection
	ExtKeyUsageIPSECEndSystem
	ExtKeyUsageIPSECTunnel
	ExtKeyUsageIPSECUser
	ExtKeyUsageTimeStamping
	ExtKeyUsageOCSPSigning
	ExtKeyUsageMicrosoftServerGatedCrypto
	ExtKeyUsageNetscapeServerGatedCrypto
)
const form = "% +v\n"

func main() {
	nargs := len(f.Args())
	if args.h || args.q || nargs == 0 {
		usage()
		os.Exit(1)
	}

	srv := f.Args()[0]

	fd, err := tls.Dial(args.n, srv, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		printerr(err)
		if fd != nil {
			printerr("bad cert")
		}
		os.Exit(1)
	}
	for _, v := range fd.ConnectionState().PeerCertificates {
		c := &Cert{Certificate: *v}
		c.KeyUsage = KeyUsage(v.KeyUsage)
		c.SignatureAlgorithm = SignatureAlgorithm(v.SignatureAlgorithm)
		c.PublicKeyAlgorithm = PublicKeyAlgorithm(v.PublicKeyAlgorithm)
		c.PublicKeyAlgorithm = PublicKeyAlgorithm(v.PublicKeyAlgorithm)
		c.Subject = Name{Name: v.Subject}
		c.Issuer = Name{Name: v.Issuer}
		data, err := json.MarshalIndent(c, "", "   ")
		if err != nil {
			printerr(err)
			os.Exit(1)
		}
		fmt.Print(string(data))
	}
	sysfatal(err)

}

type Mask *int

type Name struct {
	pkix.Name `json:",omitempty"`
	Names     Mask `json:",omitempty"`
}

type ConnState struct {
	Version                     Suite                 // TLS version used by the connection (e.g. VersionTLS12)
	CipherSuite                 Suite                 // cipher suite in use (TLS_RSA_WITH_RC4_128_SHA, ...)
	NegotiatedProtocol          string                // negotiated next protocol (from Config.NextProtos)
	NegotiatedProtocolIsMutual  bool                  // negotiated protocol was advertised by server
	ServerName                  string                // server name requested by client, if any (server side only)
	PeerCertificates            []*Cert               // certificate chain presented by remote peer
	VerifiedChains              [][]*x509.Certificate // verified chains built from PeerCertificates
	SignedCertificateTimestamps [][]byte              // SCTs from the server, if any
	OCSPResponse                []byte                // stapled OCSP response from server, if any

	// TLSUnique contains the "tls-unique" channel binding value (see RFC
	// 5929, section 3). For resumed sessions this value will be nil
	// because resumption does not include enough context (see
	// https://secure-resumption.com/#channelbindings). This will change in
	// future versions of Go once the TLS master-secret fix has been
	// standardized and implemented.
	TLSUnique []byte
}
type Cert struct {
	Raw                     Mask          `json:",omitempty"`
	RawTBSCertificate       Mask          `json:",omitempty"`
	RawSubjectPublicKeyInfo Mask          `json:",omitempty"`
	RawSubject              Mask          `json:",omitempty"`
	RawIssuer               Mask          `json:",omitempty"`
	Subject                 Name          `json:",omitempty"`
	Issuer                  Name          `json:",omitempty"`
	Extensions              Mask          `json:",omitempty"`
	PolicyIdentifiers       Mask          `json:",omitempty"`
	ExtKeyUsage             []ExtKeyUsage `json:",string"`
	KeyUsage                KeyUsage      `json:",omitempty"`
	SignatureAlgorithm      SignatureAlgorithm
	PublicKeyAlgorithm      PublicKeyAlgorithm
	x509.Certificate
}

func sysfatal(err error) {
	if err == nil {
		return
	}
	printerr(err)
	os.Exit(1)
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func printdebug(v ...interface{}) {
	if Debug {
		printerr(v...)
	}
}

func usage() {
	fmt.Println(`
NAME
	cert - certificates to your gaping maw

SYNOPSIS
	cert host:port

DESCRIPTION
	Given host and port, cert prints the certificate(s)
	presented by the host.

	A certificate with the same Subject and Issuer is
	self-signed.

EXAMPLE
	cert google.com:443
	cert yourpc:3389
`)
}
