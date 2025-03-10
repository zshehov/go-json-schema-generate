package generate

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ESOutput generates code for elasticsearch mapping and writes to w.
func ESOutput(w io.Writer, g *Generator, pkg string) {
	structs := g.Structs
	//aliases := g.Aliases

	fmt.Fprintln(w, "// Code generated by @elastic/go-json-schema-generate schema-generate quick&dirty mod for ES mappings. DO NOT EDIT.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "package %v\n\n", cleanPackageName(pkg))

	// write all the code into a buffer, compiler functions will return list of imports
	// write list of imports into main output stream, followed by the code
	codeBuf := new(bytes.Buffer)

	fmt.Fprintln(w, "const (")
	for _, k := range getOrderedStructNames(structs) {
		s := structs[k]

		var indent int

		fmt.Fprintln(w)
		outputNameAndDescriptionComment(s.Name, s.Description, w)

		fmt.Fprintf(w, "Mapping%s = `", s.Name)

		renderStructMapping(indent, w, s, structs)

		fmt.Fprintln(w, "`")

		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, ")")
	// write code after structs for clarity
	w.Write(codeBuf.Bytes())
}

func filterNoRender(structs map[string]Struct) map[string]Struct {
	result := make(map[string]Struct)
	for k, s := range structs {
		result[k] = s
	}

	for _, s := range structs {
		check(1, s, structs, result)
	}
	return result
}

func check(level int, s Struct, structs, result map[string]Struct) {
	for _, f := range s.Fields {
		esType := getESType(f)
		name := strings.Trim(f.Type, "[]*")

		if esType == "object" {
			fmt.Println("name: ", name, "level:", level)
			if s, ok := structs[name]; ok && level > 0 {
				delete(result, s.Name)
				check(level+1, s, structs, result)
			}
		}
	}
}

func renderStructMapping(indent int, w io.Writer, s Struct, structs map[string]Struct) {
	fmt.Fprintln(w, "{")
	indent++
	printIndentedLn(indent, w, `"properties": {`)

	indent++

	fieldNames := getOrderedFieldNames(s.Fields)
	var count int
	for _, fieldKey := range fieldNames {
		f := s.Fields[fieldKey]
		if strings.HasPrefix(f.JSONName, "_") || f.JSONName == "-" {
			// Do not add these fields into mapping
			// they will conflict with the ES fields
			continue
		}

		if count != 0 {
			fmt.Fprint(w, ",")
			fmt.Fprintln(w)
		}
		count++
		printIndented(indent, w, `"%s": `, f.JSONName)

		esType := getESType(f)
		name := strings.Trim(f.Type, "[]*")

		if esType == "object" {
			s, ok := structs[name]
			if ok {
				if s.AdditionalType != "" || f.Format == "raw" {
					fmt.Fprintln(w, "{")
					indent++
					printIndentedLn(indent, w, `"enabled" : false,`)
					printIndentedLn(indent, w, `"type": "%s"`, esType)
					indent--
					printIndented(indent, w, "}")
				} else {
					indent++
					renderStructMapping(indent-1, w, s, structs)
					indent--
				}
			} else {
				fmt.Fprintln(w, "{")
				indent++
				ftype := "keyword"
				if name == "int64" {
					ftype = "integer"
				}
				printIndentedLn(indent, w, `"type": "%s"`, ftype)
				indent--
				printIndented(indent, w, "}")
			}
		} else {
			fmt.Fprintln(w, "{")
			indent++
			printIndentedLn(indent, w, `"type": "%s"`, esType)
			indent--
			printIndented(indent, w, "}")
		}
	}
	printIndentedLn(indent, w, "")
	indent--
	printIndentedLn(indent, w, "}")

	indent--
	printIndented(indent, w, "}")
}

func printIndented(indent int, w io.Writer, format string, a ...interface{}) {
	if indent > 0 {
		fmt.Fprint(w, strings.Repeat("\t", indent))
	}
	fmt.Fprintf(w, format, a...)
}

func printIndentedLn(indent int, w io.Writer, format string, a ...interface{}) {
	printIndented(indent, w, format, a...)
	fmt.Fprintln(w)
}

func getESType(f Field) string {
	switch f.Type {
	case "string", "[]string":
		switch f.Format {
		case "date-time":
			return "date"
		default:
			return "keyword"
		}
	case "int":
		return "integer"
	case "bool":
		return "boolean"
	default:
		return "object"
	}
}
