# Overview

Goactors is an actor library for the Go language, useful in the development of concurrent applications.

- Based on code generation
- Patterned after Erlang's OTP gen_server behavior
- Provides a high level programming model: actors are objects with methods that perform application logic. Concurrency and message passing are handled implicitly by the library
- Offers synchronous and asynchronous messaging
- Uses standard Go syntax. Actor definition files are valid Go source files

# Getting started

An actor is a struct with methods that embeds `actor.Actor`:

```go
package hello

import (
	"carevaloc/goactor/actor"
	"fmt"
)

type hello struct {
	actor.Actor
}

func (h *hello) hello() string {
	var msg = "Hello world!"
	fmt.Println(msg)
	return msg
}
```

The structure and the methods should be unexported (lower case). You will not call these methods directly. They will be called indirectly by the actor. 

To generate the actor code you use the `actorc` tool. Assuming the hello code is in a file called hello.go and `actorc` is in your PATH:

`$ actorc -i hello.go -o hellogen.go`

The `actorc` tool will generate the actor code in the output file hellogen.go. This will be a Go source file in the same package as the input source file and should be compiled with the rest of your application.

To use the created actor:

```go
func main() {
	// create and start the actor
	h := hello.NewHello().Start()

	// make sure the actor stops when no longer needed
	defer h.Stop()

	// create a reference
	ref := h.Ref()

	// invoke the actor. Use upper case method names
	str := ref.Hello()
	fmt.Println(str)
}
```

## Asynchronous methods

Asynchronous methods are specified in a tag in the `actor.Actor` embedded field:

```Go
type calculator struct {
	actor.Actor `async:"add,mult"`
}

func (c *calculator) add(a, b int) int {
	return a + b
}
```
You may specify more than one async method separated by comas.

To call an asynchronous method and get the returned values:
```Go
	calc := calc.NewCalculator().Start()
	cref := calc.Ref()

	// Asynchronous methods return a function that can be used
	// to retrieve the results
	result := cref.Add(3, 4)

	for {
		// the last value returned will be true if the method
		// has finished and false otherwise
		if c, ok := result(); ok {
			fmt.Println(c)
			calc.Stop()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
```
If there are no return values nothing is returned.

# Generated code

The `actorc` tool generates code that turns a `struct` into an actor. The generated elements are:

* The actor interface: an interface with the name of the original struct but capitalized (exported) and the following methods:
	* Start. Starts the actor processing loop in a separate goroutine: wait for a message, execute the requested method and return the results
	* Ref. Returns an actor reference, used to invoke methods on the actor
	* Stop. Stop the actor's main loop
* Implementations of the above mentioned methods for the actor `struct`
* A New() function that creates an actor. It returns an instance of the original structure, but wrapped in the actor interface
* An actor reference with the same name as the original `struct` plus the suffix "Ref". It will have an exported method for each method in the actor `struct`
* A request and a response struct for each method

## Generated code for the hello world example

```go
// Hello is the Actor interface: an interface with the name of the original 
// struct but capitalized (exported). This will be the type returned by the
// NewHello function. The dynamic type will be a pointer to the original struct
type Hello interface {
	// Start starts the actor's main loop on a new goroutine: wait on the in
	// channel for messages, execute the requested method and return the results
	Start() Hello

	// Ref creates and returns the actor reference that will be used 
	// by client code to invoke actor methods
	Ref() *HelloRef

	// Stop stops the actor's main loop
	Stop()
}

// NewHello creates a new actor. The returned type is the actor interface
// The dynamic type is a pointer to the actor struct 
func NewHello() Hello {
	...
}

// HelloRef is the actor reference. 
type HelloRef struct {
	in     chan interface{}
	out    chan interface{}
	stopCh chan struct{}
}

// Hello sends a message to the actor's In channel requesting the execution 
// of the hello method (with the actual application logic), waits for the results
// in the out channel and returns them to the caller
func (ref *HelloRef) Hello() string {
	...
}
```
# Additional considerations

* A goactor executes tasks (methods) in its own goroutine, concurrently with other program tasks
* Tasks are executed in sequence, one after another. This guarantees that state variables (fields in the actor struct) are not accessed simultaneously from different goroutines (race conditions)
* Synchronous methods block the calling goroutine until the result is returned
* Asynchronous methods do not block the calling goroutine. These methods return immediately. The return value if any, will be a function that, when invoked, returns the results and an additional boolean value. If this value is true the method has finished and the returned values can be used. If false the method has not finished and the results will contain the zero value for their respective data types

# actorc command reference

Command `actorc` generates the necesary code to create goactors. Its input should be a Go source file with actor definitions and, possibly other code (which will be ignored). The output will be a go source file to be included in the application.

Usage:

	actorc [-v] -i input_file [-o output_file]

Options:

	-i	input file containing actor declarations. If absent, actorc will terminate with an error message.

	-o	output file. Generated actor code will be written to this file. If the file exists it will be overwritten. If absent, generated code will be sent to standard output.

	-v	verbose output for debugging.

# License

The actorc program is licensed under the GPL v3. This only applies to the source code of actorc, not the code that it generates

Goactors has an include file, actor.go that has to be imported into your program. This is licensed under the LG PL v3.