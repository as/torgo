package main

import (
	"fmt"
	"log"
	"runtime"
)

func debug(fm string, i interface{}) {
	if Debug {
		log.Printf(fm, i)
	}
}

func usage() {
	var shell = "rc"
	switch runtime.GOOS {
	case "windows":
		shell = "cmd"
	case "linux":
		shell = "sh"
	}
	fmt.Println(`
NAME
	rewire - remap file descriptors and run command

SYNOPSIS
	rewire [-fd0 cmd ... -fdN cmd] [finalcmd]

DESCRIPTION
	Rewire filters file descriptors given on the command
	line through cmd and then executes finalcmd if set.

OPTIONS
	-0 cmd     Send stdin through cmd's stdin and pipe
               cmd's stdout to the final command.

	-1 cmd     Send the output of the final command to
               cmd's stdin, output the result to stdout.

	-2 cmd     Like -1, except with stderr

	-n cmd     N is an open file descriptor and cmd is a
               command to bind to the file descriptor.

EXAMPLE
    Type in the style of a Microsoft C programmer
        ; rewire -1 'tr -d aeiou'
        ; echo hello world
		hll wrld

	Execute 'ping localhost' before every shell command
        ; rewire -0 'fm ping localhost && %s'`, shell, `
        ; echo hello world

    Encrypt stdout and decrypt stdin between remote parties:
        ; key=00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff
        ; rewire -0 'enc -r -e key' -1 'dec -r -e key' 'listen :801'

	For the client side:
        ; key=00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff
        ; rewire -0 'enc -r -e key' -1 'dec -r -e key' 'dial server:801'

BUGS
	Behavior is undefined for non-standard file descriptors,
	which are currently a specially-handled case in the program.

	If no final command is given, rewire attempts to run cat. This
	should be fixed to apply redirection to the current environment.

`)

}
