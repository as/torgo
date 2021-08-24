# Synopsis

Package log implements a structured JSON log which can't be
easily dependency injected into your microservice (on purpose).

# Variables

To use, first override the package-scoped variables at runtime.

```
var (
	// Service name
	Service = ""

	// Time is your time function
	Time    = func() interface{} {
		return time.Now().UnixNano() / int64(time.Millisecond)
	}

	// Default is the level used when calling Printf and Fatalf
	Default = Info
)
```

# Examples

main.go
```
package main

import "github.com/as/log"

func main() {
	log.Service = "ex"
	log.Time = func() interface{} { return "2121.12.04" }

	log.Error.Add(
		"env", "prod",
		"burning", true,
		"pi", 3.14,
	).Printf("error: %v", io.EOF)
}
```

output
```
{"svc":"ex", "time":"2121.12.04", "level":"error", "msg":"error: EOF", "env":"prod", "burning":true, "pi":3.14}
```

# Install

```
go get github.com/as/log
go test github.com/as/log -v -bench . 
go test github.com/as/log -race -count 1
```

This code may also be copied and pasted into your microservice
and modified to your liking. Put it in a package called
log. A little copying is better than a little dependency.
