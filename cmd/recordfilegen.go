//this program generates the appropriate record files
//based on the templates.
package main

import (
	"flag"
	"fmt"
	//"github.com/CSUNetSec/protoparse/util"
	"io"
	"os"
	"strings"
	"text/template"
)

var (
	gentype    string
	type0      string
	fmarshal   string
	funmarshal string
	pkgname    string
)

var recfiletmpl = template.Must(template.New("tmpl").Parse(
	`package {{.packagename}}

/*
   This is an autogenerated file. Do not edit directly. 
   Generator: github.com/CSUNetSec/protoparse/util/recordfilegen.go
*/

import (
	. "github.com/CSUNetSec/protoparse/util"
	"errors"
	"fmt"
	"os"
){{if .imports}}

{{range $imp := .imports}}
import (
	"{{$imp}}"{{end}}
){{end}}

var (
	errscanner = errors.New("scanner in underlying is not Open. Call Open() first")
	errind = errors.New("no such index in file")
)

type {{.typename}}RecordFiler interface {
	RecordFiler
	Put({{.type}}) (error)
}

type {{.typename}}RecordFile struct {
	*FlatRecordFile
}

func New{{.typename}}RecordFile(fname string) *{{.typename}}RecordFile {
	return &{{.typename}}RecordFile{
		NewFlatRecordFile(fname),
	}
}

func (recfile *{{.typename}}RecordFile) Put(rec {{.type}}) error {
	b := {{.fmarshal}}(rec)
	_, err := recfile.Write(b)
	if err == nil {
		recfile.entries++
	}
	return err
}

func (recfile *{{.typename}}RecordFile) Get(ind int) ({{.type}}, error) {
	if recfile.Scanner == nil {
		return {{.type0}}, errscanner
	}
	curind := 0
	for recfile.Scanner.Scan() {
		if curind == ind {
			return {{.funmarshal}}(recfile.Scanner.Bytes()), nil
		}
		curind++
	}
	return {{.type0}}, errind
}
`))

var Usage = func() {
	fmt.Fprintf(os.Stderr, ":%s [flags] outfile.go\n", os.Args[0])
	flag.PrintDefaults()
}

func init() {
	const (
		dgentype    = "string"
		dtype0      = `""`
		dfmarshal   = "[]byte"
		dfunmarshal = "string"
		dpkgname    = "main"
	)
	flag.StringVar(&gentype, "type", dgentype, "type to generate templated code for")
	flag.StringVar(&fmarshal, "fmarshal", dfmarshal, "function for marshaling that type to bytes")
	flag.StringVar(&funmarshal, "funmarshal", dfunmarshal, "function to unmarshal bytes to type")
	flag.StringVar(&pkgname, "pkgname", dpkgname, "package name for the resulting generated go file")
	flag.StringVar(&type0, "typeEmpty", dtype0, "the empty expression for the provided type")
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		Usage()
		return
	}
	if _, err := os.Stat(flag.Arg(0)); err == nil {
		fmt.Fprintf(os.Stderr, "output file already exists\n")
		return
	}
	fp, err := os.Create(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating file:%s\n", err)
		return
	}
	defer fp.Close()
	genRecordFile(fp)
}

//both funtions borrowed from timtadh/fs2/fs2-generic
func parseType(imports map[string]bool, fqtn string) string {
	ptr := ""
	if strings.HasPrefix(fqtn, "*") {
		ptr = "*"
		fqtn = strings.TrimLeft(fqtn, "*")
	}
	parts := strings.Split(fqtn, "/")
	if len(parts) == 1 {
		return ptr + fqtn
	}
	typename := ptr + strings.Join(parts[len(parts)-2:], ".")
	imp := strings.Join(parts[:len(parts)-1], "/")
	imports[imp] = true
	return typename
}

func parseFunc(imports map[string]bool, fqfn string) string {
	parts := strings.Split(fqfn, "/")
	if len(parts) == 1 {
		return fqfn
	}
	funcname := strings.Join(parts[len(parts)-2:], ".")
	imp := strings.Join(parts[:len(parts)-1], "/")
	imports[imp] = true
	return funcname
}

func genRecordFile(out io.Writer) {
	imports := make(map[string]bool)
	gentypeval := parseType(imports, gentype)
	fmarshalval := parseFunc(imports, fmarshal)
	funmarshalval := parseFunc(imports, funmarshal)
	typeparts := strings.Split(gentypeval, ".")
	justtypename := typeparts[len(typeparts)-1]
	impstrs := make([]string, 0, len(imports))
	for k := range imports {
		impstrs = append(impstrs, k)
	}
	err := recfiletmpl.Execute(out, map[string]interface{}{
		"packagename": pkgname,
		"imports":     impstrs,
		"type":        gentypeval,
		"fmarshal":    fmarshalval,
		"funmarshal":  funmarshalval,
		"typename":    justtypename,
		"type0":       type0,
	})
	if err != nil {
		fmt.Printf("error in template:%s\n", err)
	}
}