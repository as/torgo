package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/as/mute"
	"golang.org/x/crypto/ssh"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("ssh: ")
}

var arg struct {
	s, u, p, c string
	n, v       bool
	d          string
	e          string
	h          bool
	q          bool
}

var f *flag.FlagSet

type Addr struct {
	net, addr, svc string
}

func dialsplit(dial string) *Addr {
	for _, delim := range []string{"!", ":"} {
		s := strings.Split(dial, delim)
		a := &Addr{net: "tcp", svc: "22"}
		n := len(s)
		switch {
		case n == 3:
			a.svc = s[2]
			a.addr = s[1]
			a.svc = s[0]
			return a
		case n == 2:
			a.addr = s[0]
			a.svc = s[1]
			return a
		case n == 1:
			a.addr = s[0]
			return a
		}
	}
	return nil
}

// Pass is a callback that returns password from the command line argument.
// This isn't supported in many clients due to its security implications
func Pass() (string, error) {
	return arg.p, nil
}

func Signers() (signers []ssh.Signer, err error) {
	if arg.d == "" {
		return nil, fmt.Errorf("no ssh key path specified")
	}
	buf, err := ioutil.ReadFile(arg.d)
	no(err)
	d, err := ssh.ParsePrivateKey(buf)
	no(err)

	signers = append(signers, d)

	// buf, err = ioutil.ReadFile(arg.e)
	// no(err)
	// der, _ := pem.Decode(buf)
	// pub, err := x509.ParseCertificate(der.Bytes)
	// no(err)
	// e, err := ssh.NewPublicKey(pub.PublicKey)
	//no(err)
	//fmt.Println(base64.StdEncoding.EncodeToString(e.Marshal()))

	return signers, nil
}

func no(err error) {
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&arg.h, "h", false, "")
	f.BoolVar(&arg.q, "?", false, "")
	f.StringVar(&arg.s, "s", os.Getenv("ssh"), "")
	f.StringVar(&arg.u, "u", os.Getenv("user"), "")
	f.StringVar(&arg.p, "p", os.Getenv("password"), "")
	f.StringVar(&arg.d, "d", "", "")
	f.BoolVar(&arg.n, "n", false, "")
	f.BoolVar(&arg.v, "v", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
	}
}

var addr *Addr

func main() {
	addr = dialsplit(arg.s)
	if arg.s == "" || arg.h || arg.q || addr == nil {
		usage()
		os.Exit(1)
	}
	config := &ssh.ClientConfig{
		User: arg.u,
	}
	if arg.d != "" {
		config.Auth = append(config.Auth, ssh.PublicKeysCallback(Signers))
	}
	if arg.p != "" {
		config.Auth = append(config.Auth, ssh.PasswordCallback(Pass))
	} else {
		panic("no")
	}

	addrsvc := addr.addr + ":" + addr.svc
	conn, err := ssh.Dial(addr.net, addrsvc, config)
	if err != nil {
		no(err)
	}

	session, err := conn.NewSession()
	if err != nil {
		no(err)
	}
	defer session.Close()
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if len(f.Args()) > 0 {
		err = command(session, strings.Join(f.Args(), " "))
	} else {
		err = interactive(session)
	}

	if err != nil {
		if arg.v {
			no(err)
		}
		os.Exit(1)
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func interactive(session *ssh.Session) error {
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}
	err := session.RequestPty(os.Getenv("TERM"),
		atoi(os.Getenv("LINES")),
		atoi(os.Getenv("COLS")),
		modes)
	if err != nil {
		fmt.Errorf("session: %s", err)
	}
	if err := session.Shell(); err != nil {
		return fmt.Errorf("session: %s", err)
	}
	if err := session.Wait(); err != nil {
		return fmt.Errorf("session: %s", err)
	}
	return nil
}
func command(s *ssh.Session, cmd string) error {
	return s.Run(cmd)
}
func printerr(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
}
func println(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
}

func usage() {
	fmt.Println(`
NAME
	ssh - ssh client
 
SYNOPSIS
	ssh [-s host:port -u user] [-p pass | -d key.pem] [cmd]

DESCRIPTION
	Ssh connects to the host and starts the ssh
	protocol, optionally running cmd with the
	standard streams connected to the remote process.

OPTIONS
	-s host     The host in the form net!host!port
	-u user     The username
	-p pass     The password
	-d key.pem  Path to the private key (overrides password)

	If any of the above options are empty, ssh
	reads from environment variables:

	$ssh    The host
	$user   The username
	$pass   The password

EXAMPLES
	On Windows:

	Connect to 10.2.77.43 over ssh running on port 3389

	set ssh=tcp!10.2.77.43!3389
	set user=root
	set pass=insecurity
	ssh

	With command line arguments

	ssh -s tcp!10.2.77.43!3389 -u root -p insecurity
`)
}
