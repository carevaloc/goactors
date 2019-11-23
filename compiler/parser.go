package compiler

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
)

// Actor contains an actor specification extracted from a go source file
type Actor struct {
	Name    string
	Impl    string
	Methods []Method
	Init    *Method
	async   map[string]bool
}

// ExpName is the exported (uppercase) actor name
func (a *Actor) ExpName() string {
	return a.Name
}

// Ref returns the name of the actor reference
func (a *Actor) Ref() string {
	return a.Name + "Ref"
}

// Async returns true if method m is asynchronous, false otherwise
func (a *Actor) Async(m string) bool {
	return a.async[m]
}

// StopRequest returns the name of the stop request method for an actor
func (a *Actor) StopRequest() string {
	return a.Impl + "StopRequest"
}

// Package contains the specification of a Package extracted
// from a go source file
type Package struct {
	Name     string
	Imports  map[string]bool
	Actors   []*Actor
	ActorInt *ActorInterface
}

// Param contains the specification of a method parameter
type Param struct {
	Name string
	Type string
}

// Method contains an actor method specification extracted from o go
// source file
type Method struct {
	Name      string
	Params    []Param
	Async     bool
	RetValues []Param
	Comments  []string
	actor     string
}

func toLower(s string) string {
	r := []rune(s)
	s1 := strings.ToLower(string(r[0]))
	r1 := []rune(s1)
	r[0] = r1[0]
	return string(r)
}

func toUpper(s string) string {
	r := []rune(s)
	s1 := strings.ToUpper(string(r[0]))
	r1 := []rune(s1)
	r[0] = r1[0]
	return string(r)
}

// HasResponse returns true if the method returns results, false otherwise
func (m *Method) HasResponse() bool {
	if m.Async && len(m.RetVals()) == 0 {
		return false
	}
	return true
}

// RetVals returns a list of return values
func (m *Method) RetVals() []Param {
	if m.Async && len(m.RetValues) > 0 {
		return m.RetValues[0 : len(m.RetValues)-1]
	}
	return m.RetValues
}

// LName returns the lower case name of the actor
func (m *Method) LName() string {
	return toLower(m.Name)
}

// Request generates the name of the request structure for a method
func (m *Method) Request() string {
	return m.actor + m.Name + "Request"
}

// Response generates the name of the response structure for a method
func (m *Method) Response() string {
	return m.actor + m.Name + "Response"
}

// func parseComment(iter *NodeIter, nd *ast.Comment) error {
// 	text := nd.Text
// 	return nil
// }

// ActorInterface contains the information required to generate the
// actor interface. It is used to avoid using literals in the code
// generation process
type ActorInterface struct {
	New   string
	Init  string
	Start string
	Stop  string
	Ref   string
}

var actorInterface = ActorInterface{
	New:   "New",
	Init:  "init",
	Start: "Start",
	Stop:  "Stop",
	Ref:   "Ref",
}

// excludeMethods contains a list of methods that will be ignored by the generator
var excludedMethods = map[string]bool{"init": true, "InCapacity": true}

// parseStruct parses a struct in the input file and checks if it's an actor declariation.
// If it is it adds the identified actor to the actors map passed as parameter
func parseStruct(name string, t *types.Struct, actors map[string]*Actor) {
	if t.NumFields() == 0 {
		return
	}

	for i := 0; i < t.NumFields(); i++ {
		fld := t.Field(i)
		if !fld.Embedded() {
			continue
		}
		if fld.Name() == "Actor" {
			log.Printf("%s is an actor\n", name)
			act := &Actor{Name: toUpper(name), Impl: name, async: make(map[string]bool)}
			actors[name] = act

			tag := t.Tag(i)
			structTag := reflect.StructTag(tag)
			if str, ok := structTag.Lookup("async"); ok {
				var async = strings.Split(str, ",")
				for _, method := range async {
					act.async[strings.Trim(method, " \t")] = true
				}
			}
		}
	}
}

// checkImport checks if a type used in a declarion in the input file needs to be imported
// in which case it adds it to the imports map passed as parameter
func checkImport(imports map[string]bool, typeName string) {
	idx := strings.Index(typeName, ".")
	if idx == -1 {
		return
	}
	r := []rune(typeName)
	var path = string(r[:idx])

	if !imports[path] {
		imports[path] = true
	}
}

// type name separates the path from the type name
func typeName(t types.Type) string {
	s := t.String()
	idx := strings.LastIndex(s, "/")
	if idx == -1 {
		return s
	}
	r := []rune(s)
	return string(r[idx+1:])
}

// parse package parses the input file and obtains all the program types using
// the go/types conf.Check method
func parsePackage(src string) (Package, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		return Package{}, err
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("", fset, []*ast.File{f}, nil)
	if err != nil {
		return Package{}, err
	}

	log.Printf("package name: %s\n", pkg.Name())
	var actors = map[string]*Actor{}
	var imports = map[string]bool{"github.com/carevaloc/goactors/actor": true}
	var result = Package{
		Name:     pkg.Name(),
		Imports:  imports,
		ActorInt: &actorInterface,
	}

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		var t = obj.Type()
		log.Printf("Name: %s, type: %s\n", name, t)
		switch t := t.Underlying().(type) {
		case *types.Struct:
			log.Printf("struct: %s\n", obj.Name())
			parseStruct(name, t, actors)
		}
	}

	parseMethods(f, src, imports, actors, actorInterface.Init)

	log.Print("Imports: ")
	for imp := range result.Imports {
		log.Printf("Import: %s\n", imp)
	}
	log.Println()

	for _, actor := range actors {
		result.Actors = append(result.Actors, actor)
	}

	return result, nil
}

// readSrc reads the source file and returs a string with the file contents
func readSrc(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("Unable to open input file %s", fileName)
	}

	return string(b), nil
}

// ParseFile parses a go source file and creates the data structure
// that will be passed to the generator to generate the actor code
func ParseFile(fileName string) (Package, error) {
	src, err := readSrc(fileName)
	if err != nil {
		return Package{}, err
	}

	return parsePackage(src)
}

func stripFirst(s string) string {
	r := []rune(s)
	return string(r[1:])
}

// parseMethod parses the string containeng the source code read from the source file and
// visits all the function nodes. If the function is an actor method, the function signature
// is extracted, stored in a Method struct and added to the corresponding actor
func parseMethods(f *ast.File, src string, imports map[string]bool, actors map[string]*Actor, init string) {
	offset := f.Pos()
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			log.Printf("Function: %s\n", fd.Name)

			if fd.Recv == nil {
				return true
			}

			log.Printf("Function: %s, is a method\n", fd.Name)

			recv := fd.Recv
			recvType := recv.List[0].Type
			recvTypeName := src[recvType.Pos()-offset : recvType.End()-offset]
			actorName := stripFirst(recvTypeName)

			log.Printf("Receiver type: %s\n", recvTypeName)
			log.Printf("Actor name: %s\n", actorName)

			actor, ok := actors[actorName]
			if !ok {
				log.Printf("%s not found in actors map\n", actorName)
				return true
			}

			log.Println(" parameters:")

			async := actor.Async(fd.Name.Name)
			method := Method{Name: fd.Name.Name, Params: []Param{}, RetValues: []Param{}, Async: async, actor: actorName}

			for _, param := range fd.Type.Params.List {
				for _, pname := range param.Names {
					ptype := src[param.Type.Pos()-offset : param.Type.End()-offset]
					par := Param{Name: pname.Name, Type: ptype}
					method.Params = append(method.Params, par)
					checkImport(imports, ptype)
					log.Printf("  Name: %s, type: %s\n", pname, ptype)
				}
			}

			if fd.Type.Results != nil {
				log.Println(" results:")
				log.Printf("Number of results: %d\n", len(fd.Type.Results.List))
				var named = false
				for _, param := range fd.Type.Results.List {
					ptype := src[param.Type.Pos()-offset : param.Type.End()-offset]
					if len(param.Names) > 0 {
						named = true
						for _, pname := range param.Names {
							retval := Param{Name: pname.Name, Type: ptype}
							method.RetValues = append(method.RetValues, retval)
							checkImport(imports, ptype)
							log.Printf("  Name: %s, type: %s\n", pname, ptype)
						}
					} else {
						retval := Param{Type: ptype}
						method.RetValues = append(method.RetValues, retval)
						checkImport(imports, ptype)
						log.Printf("  Name: , type: %s\n", ptype)
					}
				}
				if method.Async {
					if named {
						method.RetValues = append(method.RetValues, Param{Name: "done", Type: "bool"})
					} else {
						method.RetValues = append(method.RetValues, Param{Type: "bool"})
					}
				}
			}

			if fd.Doc != nil {
				for _, comment := range fd.Doc.List {
					method.Comments = append(method.Comments, comment.Text)
					log.Println(comment.Text)
				}
			} else {
				log.Printf("Method %s has no comment\n", method.Name)
			}

			if method.Name == init {
				actor.Init = &method
			}

			_, excluded := excludedMethods[method.Name]
			if !excluded {
				method.Name = toUpper(method.Name)
				actor.Methods = append(actor.Methods, method)
			}
		}
		return true
	})
}
