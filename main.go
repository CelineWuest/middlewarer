package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/types"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	typeName = flag.String("type", "", "The interface type to wrap")
	output   = flag.String("output", "", "output file name, default srcdir/<type>_middleware.go")
)

func main() {
	flag.Parse()
	if typeName == nil {
		flag.Usage()
		log.Printf("no type name supplied")
		os.Exit(1)
	}

	g := Generator{}
	g.init(*typeName)
	fmt.Println(g.target)

	g.generateWrapperCode()

	g.print(os.Stdout)
}

type Generator struct {
	p          *packages.Package
	target     *types.Interface
	targetName string

	// Buffers for the different sections of the generated code
	wrapFunction     *bytes.Buffer
	middlewareStruct *bytes.Buffer
	handlerFuncTypes *bytes.Buffer
	interfaceMethods *bytes.Buffer
}

// init inits the generator.
// It loads the package to parse and looks for the interface
// with name matching the passed target string.
func (g *Generator) init(target string) {
	g.targetName = target

	// Load the package of the current directory
	packs, err := packages.Load(&packages.Config{
		// TODO: Make sure to minimize information here, probably getting too much
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports,
	}, ".")
	if err != nil {
		log.Printf("Failed to load packages - %v", err)
		os.Exit(1)
	}

	if len(packs) != 1 {
		log.Printf("Loaded package length is not 1, but %d", len(packs))
		os.Exit(1)
	}
	g.p = packs[0]

	// Look for the matching interface
	obj := g.p.Types.Scope().Lookup(target)
	if obj == nil {
		log.Fatalf("Couldn't find target object '%s' in source file", target)
	}

	iFace, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		log.Fatalf("Provided target object '%s' is not an interface", target)
	}

	g.target = iFace
}

const wrapFunctionFormat = `// Wrap%[1]s returns the passed %[1]s wrapped in the middleware defined in %[2]s
func Wrap%[1]s(toWrap %[1]s, wrapper %[2]s) %[1]s {
	wrapper.wrapped = toWrap
	return &wrapper
}
`

// generateWrapperCode generates the code for the wrapper of the target interface
func (g *Generator) generateWrapperCode() {
	g.wrapFunction = new(bytes.Buffer)
	g.middlewareStruct = new(bytes.Buffer)
	g.handlerFuncTypes = new(bytes.Buffer)
	g.interfaceMethods = new(bytes.Buffer)

	structName := fmt.Sprintf("%sMiddleware", g.targetName)

	// Write wrap function
	fmt.Fprintf(g.wrapFunction, wrapFunctionFormat, g.targetName, structName)
}

// print writes the generated code to the provided io.Writer
func (g *Generator) print(w io.Writer) {
	// Print header
	fmt.Fprintf(w, "// Code generated by \"middlewarer %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	fmt.Fprintf(w, "package %s\n", g.p.Name)
	fmt.Fprintln(w)

	// Print the generated code
	w.Write(g.wrapFunction.Bytes())
	fmt.Fprintln(w)
	w.Write(g.middlewareStruct.Bytes())
	fmt.Fprintln(w)
	w.Write(g.handlerFuncTypes.Bytes())
	fmt.Fprintln(w)
	w.Write(g.interfaceMethods.Bytes())
	fmt.Fprintln(w)
}
