package compiler

import (
	"io"
	"log"
	"text/template"
)

// Generate generates the actor code passing a Packate
// object containig actor definitions to a text/Template
func Generate(output io.Writer, pkg Package) {
	funcMap := template.FuncMap{
		"toLower": toLower,
		"toUpper": toUpper,
	}

	t := template.New("Actor template").Funcs(funcMap)

	t, err := t.Parse(actorTmpl)
	if err != nil {
		log.Fatal("Parse: ", err)
	}

	err = t.Execute(output, pkg)
}

// actorTmpl is a template (/text/Template) used to generate the actor code
const actorTmpl = `package {{.Name}}
{{$actorInt := .ActorInt}}
import (
{{- range $key, $value := .Imports}}
	"{{$key}}"
{{- end}}
)
{{range .Actors}}{{$actorName := .ExpName}}{{$actorRef := .Ref}}{{$actorImpl := .Impl}}{{$methods := .Methods}}{{$init := .Init}}{{$stopRequest := .StopRequest}}
type {{$actorName}} interface {
	Start() {{$actorName}}
	Ref() *{{$actorRef}}
	Stop()
}

type {{$actorRef}} struct {
	in  chan interface{}
	out chan interface{}
	stopCh chan struct{}	
}

func {{$actorInt.New}}{{$actorName}}({{if $init}}{{- range $i, $param:=$init.Params}}{{if $i}}, {{end}}{{- .Name}} {{.Type}}{{end}}{{end}}) {{$actorName}} {
	act := &{{$actorImpl}} {
		Actor: actor.Actor{},
	}
	act.In = make(chan interface{}, act.InCapacity())
	act.StopCh = make(chan struct{})
{{- if $init}}
	act.{{$actorInt.Init}}({{- range $i, $param:=$init.Params}}{{if $i}}, {{end}}{{- .Name}}{{end}})
{{- end}}	
	return act
}

func (act *{{$actorImpl}}) {{$actorInt.Start}}() {{$actorName}} {
	go act.receive()
	return act
}

func (act *{{$actorImpl}}) {{$actorInt.Ref}}() *{{$actorRef}} {
	ref := &{{$actorRef}}{
		in:  act.In,
		stopCh: act.StopCh,		
		out: make(chan interface{}),
	}
	return ref
}

func (ref *{{$actorRef}}) Stopped() bool {
	select {
	case <-ref.stopCh:
		return true
	default:
		return false
	}
}

type {{$stopRequest}} struct {}

func (act *{{$actorImpl}}) Stop() {
	act.In <- {{$stopRequest}}{}
}
{{range .Methods}}{{$met := .}}
{{$params := $met.Params}}{{$retValues := $met.RetValues -}}
type {{$met.Request}} struct {
	ref *{{$actorRef}}
{{range $params}}	{{.Name}} {{.Type}} 
{{end -}} }

type {{$met.Response}} struct {
{{range $i, $retVal := $met.RetVals}} r{{$i}} {{.Type}}
{{end -}} }

{{range $i, $comment := $met.Comments}}
{{$comment}}
{{end -}}
func (ref *{{$actorRef}}) {{$met.Name}}(
{{- range $i, $param:=$met.Params}}{{if $i}}, {{end}}{{- .Name}} {{.Type}}{{end}})
{{- if $retValues}} {{- if $met.Async}} func(){{end}} (
{{- range $i, $ret:=$retValues}}{{- if $i}}, {{end}}{{if .Name}}{{- .Name}} {{end}}{{.Type}}{{end}})
{{- end}} {
	select {
	case <-ref.stopCh:
		panic("Actor stopped")
	default:
	}
	select {
	case ref.in <- {{$met.Request}}{ref{{if $met.Params}}, {{- range $i, $param:=$met.Params}}{{if $i}}, {{end}}{{- $param.Name}}{{- end}}{{end}}}:
{{- if $retValues}}
{{- if $met.Async}}
		return func() {{if $retValues -}}
			({{- range $i, $ret:=$retValues}}{{if $i}}, {{end}}{{.Type}}{{end}}){{end}} {
			select {
			case result := <-ref.out:
				if result, ok := result.({{$met.Response}}); ok {
					return {{range $i, $ret := $met.RetVals}}{{if $i}}, {{end}}result.r{{$i}}{{end}}, true
				}
				panic("Wrong type of result message received")			
			default:
				result := {{$met.Response}}{}
				return {{range $i, $ret := $met.RetVals}}{{if $i}}, {{end}}result.r{{$i}}{{end}}, false
			}
		}
	}
{{- else}}
		result := <-ref.out
		if result, ok := result.({{$met.Response}}); ok {
			return {{range $i, $ret := $met.RetValues}}{{if $i}}, {{end}}result.r{{$i}}{{end}}
		}
		panic("Wrong type of result message received")
	default:
		panic("Unknown error")
	}
{{- end}}
{{else}}
{{- if not $met.Async}}
		<-ref.out
{{end -}}
	}
{{end -}} }
{{end}}
func (act *{{$actorImpl}}) receive() {
	var stopped = false
	var msg interface{}
	for {
		if !stopped {
			msg = <-act.In
		} else {
			select {
			case msg = <-act.In:
			default:
				actor.Log.Println("No more messages. Exiting")
				return
			}
		}
		switch msg := msg.(type) {
{{- range $methods}}{{$met:=.}}{{$retVals:=$met.RetVals}}
		case {{$met.Request}}:
			{{range $i, $ret := $retVals}}{{if $i}}, {{end}}v{{$i}}{{end}}{{if $retVals}} := {{end -}}
			act.{{$met.LName}}(
{{- range $i, $param:=$met.Params}}{{if $i}}, {{end}}msg.{{- $param.Name}}{{end}})
{{- if $met.HasResponse}}
			msg.ref.out <- {{$met.Response}}{ {{- range $i, $ret:=$retVals}}{{if $i}}, {{end}}v{{$i}}{{end}}}
{{- end}}
{{- end}}
		case {{$stopRequest}}:
			close(act.StopCh)
			stopped = true
			actor.Log.Println("Actor stopped")
		default:
			msg = msg
			panic("Wrong type of request message received")			
		}
	}
}
{{end}}
`

const printActor = `package stack
{{range .}}
Actor {{.Name}}
{{range .Methods}}
	Method: {{.Name}} ({{range .Params}} {{.Name}} {{.Type}}, {{end}}) {{range .RetValues}} {{.Name}} {{.Type}} {{end}} {{end}}
{{end}}
`
