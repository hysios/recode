package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	src    string
	label  string
	row    string
	col    string
	colSep string
	input  string
)

func main() {
	parseArgs()
	fs := token.NewFileSet()

	r, err := os.Open(src)
	if err != nil {
		fmt.Println(err)
		return
	}

	rd, err := openInput()
	if err != nil {
		fmt.Println(err)
	}

	source, err := io.ReadAll(r)
	if err != nil {
		fmt.Println(err)
		return
	}
	r.Close()

	f, err := parser.ParseFile(fs, src, string(source), parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	var (
		comments = lookupCommentLabel(label, f)
		code     = ""
	)

	if row != "" {
		code = generateRowOutput(row, rd)
	} else if col != "" {
		code = generateColOutput(col, colSep, rd)
	} else {
		code = generateRowOutput("{{ . }}", rd)
	}
	fmt.Printf("code: %s\n", code)
	if len(code) > 0 {
		code += "\n"
	}
	if len(comments) == 2 && comments[1].Pos() > comments[0].End() {
		// modify code
		fmt.Printf("start %v end %v\n", comments[0].End(), comments[1].Pos())
		// code = insertCode(string(source), comments[1].Pos(), code)
		code = replaceCode(string(source), comments[0].End(), comments[1].Pos(), code)
		writeCode(src, code)
		formatCode(src)
	}
}

func lookupCommentLabel(label string, f *ast.File) []*ast.Comment {
	var comments []*ast.Comment
	for _, c := range f.Comments {
		for _, cl := range c.List {
			line := strings.TrimPrefix(cl.Text, "//")
			line = strings.TrimSpace(line)

			fmt.Printf("line: %s %s\n", line, label)
			if strings.TrimSuffix(line, "-BEGIN") == label {
				comments = append(comments, cl)
				fmt.Println("begin")
				// break
			} else if strings.TrimSuffix(line, "-END") == label {
				comments = append(comments, cl)
				fmt.Println("end")
				// break
			}

		}
	}

	return comments
}

func parseArgs() {
	flag.StringVar(&src, "src", "", "source file")
	flag.StringVar(&label, "label", "", "label of comments for modify code")
	flag.StringVar(&input, "input", "", "input file")
	flag.StringVar(&row, "row", "", "generate row output")
	flag.StringVar(&col, "col", "", "generate col output")
	flag.StringVar(&colSep, "sep", ",", "col sep")

	flag.Parse()

	if src == "" {
		src = os.Getenv("GOFILE")
	}

	if label == "" {
		flag.Usage()
		os.Exit(1)
	}
}

// openInput
func openInput() (io.Reader, error) {
	if input == "" {
		return os.Stdin, nil
	}

	return os.Open(input)
}

// generateRowOutput generate row output
func generateRowOutput(row string, rd io.Reader) string {
	// build row template
	var (
		tpl = template.Must(template.New("row").Funcs(funcs).Parse(row))
		s   = bufio.NewScanner(rd)
	)

	s.Split(bufio.ScanLines)

	var lines = make([]string, 0)
	for s.Scan() {
		var sb strings.Builder
		fmt.Printf("text: %s\n", s.Text())
		if err := tpl.Execute(&sb, s.Text()); err != nil {
			log.Printf("tpl execute error: %v\n", err)
		}
		lines = append(lines, sb.String())
	}

	return strings.Join(lines, "\n")
}

// generateColOutput generate col output
func generateColOutput(col, sep string, rd io.Reader) string {
	var (
		tpl = template.Must(template.New("col").Funcs(funcs).Parse(col))
		s   = bufio.NewScanner(rd)
	)

	s.Split(bufio.ScanLines)

	var lines = make([]string, 0)
	for s.Scan() {
		var sb strings.Builder
		tpl.Execute(&sb, s.Text())
		lines = append(lines, sb.String())
	}

	return strings.Join(lines, sep)
}

// writeCode write code to file
func writeCode(src string, code string) error {
	f, err := os.OpenFile(src, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.WriteString(code); err != nil {
		return err
	}

	return nil
}

// formatCode format code
func formatCode(src string) error {
	return exec.Command("goimports", "-w", src).Run()
}

// insertCode
func insertCode(src string, pos token.Pos, code string) string {
	var (
		content = []byte(src)
		p       = int(pos)
	)

	var sb strings.Builder
	sb.Grow(len(content) + len(code))
	sb.Write(content[:p-2])
	sb.WriteString(code)
	sb.Write(content[p-2:])

	return sb.String()
}

// replaceCode replace code
func replaceCode(src string, start, end token.Pos, code string) string {
	var (
		content = []byte(src)
		s       = int(start)
		e       = int(end)
	)

	var sb strings.Builder
	sb.Grow(len(content) + len(code))
	sb.Write(content[:s])
	sb.WriteString(code)
	sb.Write(content[e-2:])

	return sb.String()
}

var funcs = template.FuncMap{
	"dirname": func(path string) string {
		return filepath.Dir(path)
	},
	"basename": func(path string) string {
		return filepath.Base(path)
	},
	"ext": func(path string) string {
		return filepath.Ext(path)
	},
	"join": func(elem ...string) string {
		return filepath.Join(elem...)
	},
	"split": func(path string, sep ...string) []string {
		var s = " "
		if len(sep) > 0 {
			s = sep[0]
		}
		log.Printf("split: %s %v\n", path, sep)
		return strings.Split(path, s)
	},
	"trim": func(s string, cutset string) string {
		return strings.Trim(s, cutset)
	},
	"trimPrefix": func(s string, prefix string) string {
		return strings.TrimPrefix(s, prefix)
	},
	"trimSuffix": func(s string, suffix string) string {
		return strings.TrimSuffix(s, suffix)
	},
	"strip": func(s string) string {
		return strings.TrimSpace(s)
	},
	"lower": func(s string) string {
		return strings.ToLower(s)
	},
	"upper": func(s string) string {
		return strings.ToUpper(s)
	},
}
