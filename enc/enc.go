package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	//	"crypto/rc4"
	"encoding/hex"
)
import (
	"github.com/as/mute"
	"github.com/as/pkcs7"
)

const (
	Prefix = "enc: "
	Unset  = "‡"
)

var (
	Role     = Prefix[:3]
	Opposite = func() string {
		if Role == "dec" {
			return "enc"
		}
		return "dec"
	}()
	NBuffers  = 0
	Streaming = false
	BlockSize = 16
)

var args struct {
	h, q    bool
	r       bool
	a, s, m string
	l       int
	k, f, e string
	i       string
	p       string
}

var f *flag.FlagSet

type Std string
type Mode string
type Keylen string
type algorithm struct {
	Std
	Mode
	Keylen
}

type blockstream struct {
	cipher.Stream
}

type ECBEncrypter struct {
	cipher.Block
}

type ECBDecrypter struct {
	cipher.Block
}

func Blockify(s cipher.Stream) *blockstream {
	return &blockstream{s}
}
func (s *blockstream) BlockSize() int {
	return BlockSize
}
func (s *blockstream) CryptBlocks(dst, src []byte) {
	s.XORKeyStream(dst, src)
}

func NewCBC(block cipher.Block, iv []byte) (cb cipher.BlockMode) {
	if Role == "enc" {
		return cipher.NewCBCEncrypter(block, iv)
	}
	return cipher.NewCBCDecrypter(block, iv)
}
func NewECB(block cipher.Block, iv []byte) (cb cipher.BlockMode) {
	if Role == "enc" {
		return NewECBEncrypter(block, iv)
	}
	return NewECBDecrypter(block, iv)
}
func NewCFB(block cipher.Block, iv []byte) (cb cipher.BlockMode) {
	if Role == "enc" {
		return Blockify(cipher.NewCFBEncrypter(block, iv))
	}
	return Blockify(cipher.NewCFBDecrypter(block, iv))
}
func NewCTR(block cipher.Block, iv []byte) (cb cipher.BlockMode) {
	return Blockify(cipher.NewCTR(block, iv))
}
func NewOFB(block cipher.Block, iv []byte) (cb cipher.BlockMode) {
	return Blockify(cipher.NewOFB(block, iv))
}

// NewECBEncrypter implements ECB encryption. This is for
// demonstration purposes only. ECB leaks information about
// the plaintext because given enc(M¹, K) → C¹  enc(M², K) → C²
// C¹ = C² if M¹ = e²
func NewECBEncrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	return &ECBEncrypter{b}
}

func (e *ECBEncrypter) CryptBlocks(dst, src []byte) {
	bs := e.BlockSize()
	for len(src) > 0 {
		e.Encrypt(dst[:bs], src[:bs])
		dst, src = dst[bs:], src[bs:]
	}
}

// NewECBDecrypter implements ECB decryption.
func NewECBDecrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	return &ECBDecrypter{b}
}

func (e *ECBDecrypter) CryptBlocks(dst, src []byte) {
	bs := e.BlockSize()
	for len(src) > 0 {
		e.Decrypt(dst[:bs], src[:bs])
		dst, src = dst[bs:], src[bs:]
	}
}

func (s Std) SetCipher(key []byte) (c cipher.Block, err error) {
	switch s {
	case "":
		return aes.NewCipher(key)
	case "aes":
		return aes.NewCipher(key)
	case "des":
		return des.NewCipher(key)
	case "des3":
		return des.NewTripleDESCipher(key)
	default:
		return nil, fmt.Errorf("cipher not found: %s", s)
	}
}

func (m Mode) SetMode(block cipher.Block, iv []byte) interface{} {
	switch m {
	case "cbc":
		return NewCBC(block, iv)
	case "ecb":
		return NewECB(block, iv)
	case "ctr":
		Streaming = true
		return cipher.NewCTR(block, iv)
	case "ofb":
		Streaming = true
		return cipher.NewOFB(block, iv)
	}

	return nil
}

func Alg(std, mode string, key []byte) (opmode interface{}, err error) {
	a := &algorithm{
		Std(std),
		Mode(mode),
		Keylen(len(key)),
	}
	if bits := len(key) * 8; bits != args.l {
		return nil, fmt.Errorf("bad key length for %s: %d", args.s, bits)
	}
	block, err := a.SetCipher(key)
	if err != nil {
		return nil, err
	}
	BlockSize = block.BlockSize()
	iv := mustiv(BlockSize)
	return a.SetMode(block, iv), nil
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.StringVar(&args.a, "a", "aes/cbc/256", "")
	f.StringVar(&args.s, "s", "", "")
	f.StringVar(&args.m, "m", "", "")
	f.IntVar(&args.l, "l", 0, "")
	// Allow someone to have an empty key by making the default
	// something else.
	f.StringVar(&args.k, "k", Unset, "")
	f.StringVar(&args.f, "f", Unset, "")
	f.StringVar(&args.e, "e", Unset, "")
	f.StringVar(&args.p, "p", Unset, "")

	f.BoolVar(&args.r, "r", false, "")
	f.StringVar(&args.i, "i", "", "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func operation(s string) func(io.Writer, io.Reader, cipher.BlockMode) (int, error) {
	switch s {
	case "enc":
		return encrypt
	case "dec":
		return decrypt
	}
	printerr("internal bug: report to author")
	os.Exit(1)
	return nil
}

func main() {
	if args.h || args.q {
		usage()
	}
	var (
		err error
		w   = os.Stdout
		r   = os.Stdin
	)

	mustparsealg()
	alg, err := Alg(args.s, args.m, mustkey())
	if err != nil {
		printerr(err)
		os.Exit(1)
	}

	switch t := alg.(type) {
	case cipher.BlockMode:
		_, err = operation(Role)(w, r, t)
	case cipher.Stream:
		_, err = stream(w, r, t)
	}

	if err != nil && err != io.EOF {
		printerr(err)
		os.Exit(1)
	}
}

func randinject(r io.Reader, n int) io.Reader {
	rb := make([]byte, BlockSize)
	rand.Read(rb)
	return io.MultiReader(bytes.NewReader(rb), r)
}

func stream(w io.Writer, r io.Reader, s cipher.Stream) (int64, error) {
	if args.r && Role == "enc" {
		r = randinject(r, BlockSize)
	}
	sr := cipher.StreamReader{s, r}
	if args.r && Role == "dec" {
		if n, err := io.CopyN(ioutil.Discard, sr, int64(BlockSize)); err != nil {
			return n, err
		}
	}
	return io.Copy(w, sr)
}

func encrypt(w io.Writer, r io.Reader, alg cipher.BlockMode) (int, error) {
	var (
		err error
		p   []byte
		n   int
		b   = make([]byte, BlockSize)
	)
	if args.r {
		r = randinject(r, BlockSize)
	}
	for err == nil {
		n, err = r.Read(b)
		if n < 1 {
			continue
		}
		p, err = pkcs7.Pad(b[:n], BlockSize)
		if len(p) < BlockSize {
			continue
		}
		alg.CryptBlocks(p, p)
		n, err = w.Write(p)
	}
	return n, err
}
func decrypt(w io.Writer, r io.Reader, alg cipher.BlockMode) (int, error) {
	var (
		err error
		p   []byte
		n   int
		b   = make([]byte, BlockSize*2)
	)
	for err == nil {
		n, err = r.Read(b)
		if err != nil {
			break
		}
		if n < BlockSize {
			continue
		}
		alg.CryptBlocks(b[:n], b[:n])
		if p, err = pkcs7.Unpad(b[:n], BlockSize); err != nil {
			break
		}
		if args.r {
			args.r = false
			continue
		}
		n, err = w.Write(p)
	}
	return n, err
}

//
// Goroutines
//

type ModReader struct {
	mod int
	r   io.Reader
	buf *bytes.Buffer
}

func newModReader(r io.Reader, m int) *ModReader {
	return &ModReader{
		m,
		r,
		new(bytes.Buffer),
	}
}
func (m ModReader) Read(p []byte) (int, error) {
	maxread := int64(len(p))
	lr := io.LimitReader(m.r, maxread)
	n, _ := m.buf.ReadFrom(lr)
	if n == 0 {
		return 0, io.EOF
	}
	align := m.buf.Len()
	if int(n) > m.mod {
		align -= align % m.mod
	}
	printerr("modreader: Read: m.mod", m.mod)
	printerr("modreader: Read: (align)", align)
	return io.ReadFull(m.buf, p[:align])

}

type MinReader struct {
	min int
	r   io.Reader
}

func newMinReader(r io.Reader, m int) *MinReader {
	return &MinReader{m, r}

}
func (m MinReader) Read(p []byte) (n int, err error) {
	return io.ReadAtLeast(m.r, p, m.min)
}

//
// Helper
//

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func dieon(err error) {
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func mustkey() (key []byte) {
	n := 0
	for _, a := range []string{args.k, args.e, args.f} {
		if a != Unset {
			n++
		}
	}
	if n != 1 {
		dieon(fmt.Errorf("key: not provided"))
	}

	switch {
	case n > 1:
		dieon(fmt.Errorf("key: too many keys"))
	case args.e != Unset:
		return musthex(os.Getenv(args.e))
	case args.f != Unset:
		fkey, err := ioutil.ReadFile(args.f)
		dieon(err)
		return musthex(string(fkey))
	case args.k != Unset:
		return musthex(args.k)
	}
	dieon(fmt.Errorf("key: not provided"))
	return nil
}

func mustiv(bs int) (iv []byte) {
	iv = musthex(args.i)
	if len(iv) == 0 {
		iv = make([]byte, bs)
	}
	if bs != len(iv) {
		dieon(fmt.Errorf("iv: bad length: %d: need blocksize: %d", bs, len(iv)))
	}
	return iv
}

// checkalgs validates the encryption algorithm argument.
// The standard, operating mode, and block length are validated
// and their parametric values are placed inside args.l, args.m,
// and args.s.
func mustparsealg() {
	keys := [...]interface{}{&args.s, &args.m, &args.l}
	vals := strings.FieldsFunc(args.a, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	nv := len(vals)
	if nv != 3 {
		dieon(fmt.Errorf("garbled algorithm: %s", args.a))
	}
	for i, ptr := range keys {
		if ptr == nil || i > nv {
			break
		}
		switch t := ptr.(type) {
		case *int:
			*t = mustatoi(vals[i])
		case *string:
			*t = vals[i]
		}
	}
}

func musthex(a string) (h []byte) {
	h = []byte(a)
	hexlen, err := hex.Decode(h, h)
	dieon(err)
	return h[:hexlen]
}

func mustatoi(a string) int {
	i, err := strconv.Atoi(a)
	dieon(err)
	return i
}

func usage() {
	fmt.Printf(`
NAME
	%[1]s - %[1]ssrypt messages

SYNOPSIS
	%[1]s [-a alg] [-r | -i iv ] -e keyvar
	%[1]s [ options ] -f keyfile
	%[1]s [ options ] -k key

DESCRIPTION

	DO NOT USE THIS PROGRAM FOR SERIOUS CRYPTOGRAPHIC WORK

	%s encrypts plaintext from stdin and outputs resulting
	ciphertext to stdout. It uses the encryption alogrithm alg
	and padding algorithm PKCS#7.

	An encryption algorithm consists of a encryption standard,
	block size, and operating mode. 

	-a alg	Use algorithm alg. The default is aes/cbc/256.	
			Mode defaults to cbc for block ciphers and default 
			blocksize for std.

	The next options provide the encryption key to %s. The key is
	hex encoded. Whitespace is truncated, any other runes are
	invalid.

	-e var   Var names the environment variable holding the key. 
	-k key   Key is the key itself
	-f file  File names the file containing the key on the first line.

	Options for semantic security. Ignored for ebc mode
	and stream ciphers. 

	-r       Random block is prepended to the plaintext and encrypted
	-i iv    Initialization vector (hex encoded)

ALGORITHMS
	Only aes, des, and des3 is enabled at this time.

	aes       Advanced Encryption Standard
	des       Data Encryption Standard
	des3      Triple DES

	otp       One-time pad
	blowfish  Blowfish
	cast5     CAST5
	salsa     Salsa
	tea       Tea
	twofish   Twofish
	xtea      Xtea

BLOCK MODES
	Only cbc and ecb is enabled at this time. Using ecb provides
	you with zero security. Use at your own risk.

	cbc       Cipher Block Chaining	
	ecb       Elecrtonic Code Book

	cbf       Cipher Feedback
	crt       Counter
	ofb       Output Feedback

EXAMPLE
	Encrypt plaintext m with default aes/cbc/256. Prepend one
	random block of noise to the plaintext before encrypting to
	randomize the block chain.

		# Note how kv is used (kv != $kv)
		# The env variable's name (not value)
		# is seen on the command line.

		echo -n we crash at dawn > m
		kv=000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f
		bvghhhhhh-r -e kv < m

	Encrypt m with aes/cbc/128. Key is set accordingly.

		kv=00112233445566778899aabbccddeeff
		enc -a aes/cbc/128 -e kv < m

BUGS
	No GCM

	ECB mode produces ciphertext that reveals patterns in
	the underlying plaintext:

		enc(B¹, K) → C¹ 
		enc(B², K) → C²
		C¹ = C² if B¹ = B²

	Enc and dec do not authenticate unless a standard operating
	mode supports bundled authentication. Use HMAC and friends
	to implement authentication alongsize enc and dec.
	

SEE ALSO
	%[2]s
	Gen
	Hmac
`, Role, Opposite)
	os.Exit(0)
}
