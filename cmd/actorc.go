package main

import (
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/carevaloc/goactors/actor"
	"github.com/carevaloc/goactors/compiler"
)

var act actor.Actor

func main() {
	input := flag.String("i", "", "input file")
	output := flag.String("o", "", "output file")
	verbose := flag.Bool("v", false, "verbose console output (for debbuging)")

	flag.Parse()

	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("input file: %s\n", *input)
	log.Printf("output file: %s\n", *output)

	if *input == "" {
		fmt.Println("No input file specified")
		os.Exit(1)
	}

	if *output == *input {
		fmt.Println("Input file and output file are the same")
		os.Exit(2)
	}

	actors, err := compiler.ParseFile(*input)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(3)
	}

	var bldr strings.Builder

	compiler.Generate(&bldr, actors)

	src, err := format.Source([]byte(bldr.String()))
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(4)
	}

	var out *os.File
	if *output == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(*output)
		if err != nil {
			fmt.Printf("Unable to create output file %s\n", *output)
			os.Exit(5)
		}
		// fmt.Printf("Writing output to %s\n", *output)
	}

	// compiler.Generate(out, actors)

	out.Write(src)
}
