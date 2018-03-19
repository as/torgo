// Rewire manipulates file descriptors by filtering them
// through auxillary shell commands. It then starts a final
// process with those redirects applied.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Debug controls verbose debug output
const Debug = false

// DevNull represents /dev/null or nul, depending on runtime.GOOS
var DevNull *os.File

func init() {
	var err error
	if DevNull, err = os.Open(os.DevNull); err != nil {
		log.Fatal(err)
	}
	log.SetFlags(0)
	log.SetPrefix("rewire: ")
}

// Cmd wraps *exec.Cmd and keeps track of which
// file descriptors have been set for the command
type Cmd struct {
	*exec.Cmd
	wasset [3]bool
}

// Redirect stages redirection of a file descriptor
// to and from a remote command. If fd is 0, or 1 the
// behavior is specifically-defined:
//
// fd = 0: r read from stdin,  pipe r's stdout to c's stdin
// fd = 1: r writes to stdout, pipe r's stdin to c's stdout
//
func (c *Cmd) Redirect(r *exec.Cmd, fd uintptr) {
	file := os.NewFile(fd, "")
	if file == nil {
		log.Fatal("bad file descriptor", fd)
	}
	switch fd {
	case 0:
		r.Stdin = os.Stdin
		os.Stdin = DevNull
		c.Stdin, _ = r.StdoutPipe()
	case 1:
		r.Stdout = os.Stdout
		os.Stdout = DevNull
		r.Stdin, _ = c.StdoutPipe()
	case 2:
	default:
		delta := fd - uintptr(len(c.ExtraFiles)) - 2
		if delta > 0 {
			c.ExtraFiles = append(c.ExtraFiles, make([]*os.File, delta)...)
		}
		c.ExtraFiles[fd-3] = file
	}
	if fd < 3 {
		// We'll look at this before running the final command
		// to see if we should set any unset fd's to this process's
		// standard descriptors.
		c.wasset[fd] = true
	}
}

// Start runs the command, it sets any unset
// file descriptors to rewire's standard file
// descriptors.
func (c *Cmd) Start() error {
	if !c.wasset[0] {
		c.Stdin = os.Stdin
	}
	if !c.wasset[1] {
		c.Stdout = os.Stdout
	}
	if !c.wasset[2] {
		c.Stderr = os.Stderr
	}
	return c.Cmd.Start()
}

// CmdParse splits the argument into fields, and the
// fields are tokenized into a new command and arguments.
func CmdParse(args ...string) (*exec.Cmd, error) {
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("no command given", args)
	case 1:
		debug("command is: %q\n", args[0])
		return exec.Command(args[0]), nil
	default:
		debug("command is: %q\n", args)
		return exec.Command(args[0], args[1:]...), nil
	}
}

func main() {
	args := checkarg(os.Args)
	parsed := Parse(args)
	if parsed == nil {
		log.Fatal("unexpected internal error")
	}

	//
	// Need to know when commands terminate early.
	//
	var wg sync.WaitGroup    // knows when everything exits
	done := make(chan error) // recieves errors until closed

	if len(parsed.finalcmd) == 0 {
		parsed.finalcmd = []string{"cat"}
	}
	// Create the final command, but don't run it yet
	m, err := CmdParse(parsed.finalcmd...)
	if err != nil {
		log.Fatal(err)
	}
	final := &Cmd{Cmd: m}

	// Apply every redirect specified on the command line. At this
	// time, we will only mention stdout and stdin in the manual. The
	// redirects are applied to the final command before it is executed.
	for _, r := range parsed.redirs {
		cmd, err := CmdParse(strings.Fields(r.cmd)...)
		if err != nil {
			log.Fatal(err)
		}
		final.Redirect(cmd, r.fd)
		if err = cmd.Start(); err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		Waiter(cmd, done)
	}

	// Wait for all members of the group to finish. Then
	// close the done channel.
	go func() {
		wg.Wait()
		close(done) // Terminates program
	}()

	// Start the final command and wait for it to terminate. When
	// it terminates, the redirector commands terminate too, since
	// this program exits.
	//
	// Is it always desirable for this to happen?
	//
	go func() {
		err = final.Start()
		if err != nil {
			log.Fatal(err)
		}
		err = final.Wait()
		if err != nil {
			log.Fatal(err)
		}
		close(done) // Terminates program
	}()

	for e := range done {
		if e != nil {
			log.Println("recieved err:", e)
		}
		wg.Done()
	}

	// Fin
}

// Waiter runs the cmd. When the cmd is finished
// waiter sends the error value to the done channel
func Waiter(cmd *exec.Cmd, done chan error) {
	go func() {
		err := cmd.Wait()
		done <- err
	}()
}

// checksarg validates the input arguments and
// terminates the process if they are invalid.
func checkarg(A []string) []string {
	if len(A) <= 1 {
		no(fmt.Errorf("usage: rewire -0 cmd ... [-n cmd]"))
	}
	if A[1] == "-h" {
		usage()
		os.Exit(0)
	}
	return A[1:]
}
