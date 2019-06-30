// go:generate ./gen_template_consts.sh
package lang

import (
	"fmt"
	"io"
	"os/exec"
	"text/template"

	"github.com/linux4life798/gobf/gobflib/il"
)

const (
	DefaultDataSize = 100000
)

type TemplateParams struct {
	InitialDataSize int
	Body            <-chan string
}

func ilBlockGo(b *il.ILBlock, cout chan<- string) {
	if b == nil {
		cout <- ""
		return
	}

	switch b.GetType() {
	case il.ILList:
		for _, ib := range b.GetInner() {
			ilBlockGo(ib, cout)
		}
	case il.ILLoop:
		cout <- "for data[datap] != 0 {\n"
		for _, ib := range b.GetInner() {
			ilBlockGo(ib, cout)
		}
		cout <- "}\n"
	case il.ILDataPtrAdd:
		cout <- fmt.Sprintf("datapadd(%d)\n", b.GetParam())
	case il.ILDataAdd:
		delta := byte(b.GetParam())
		cout <- fmt.Sprintf("dataadd(%v)\n", delta)
	case il.ILRead:
		for i := int64(0); i < b.GetParam(); i++ {
			cout <- "readb()\n"
		}
	case il.ILWrite:
		cout <- fmt.Sprintf("writeb(%v)\n", b.GetParam())
	}
}

func ILBlockToGo(b *il.ILBlock, output io.Writer) error {
	var useGoFmt bool = true
	var err error

	_, err = exec.LookPath("gofmt")
	if err != nil {
		useGoFmt = false
	}

	var c = make(chan string, 1024)
	go func() {
		ilBlockGo(b, c)
		close(c)
	}()

	var params = TemplateParams{
		InitialDataSize: DefaultDataSize,
		Body:            c,
	}
	t := template.Must(template.New("main").Parse(templateConstMain))

tryagain:
	if useGoFmt {
		gofmt := exec.Command("gofmt")
		pinput, err := gofmt.StdinPipe()
		if err != nil {
			pinput.Close()
			useGoFmt = false
			goto tryagain
		}
		gofmt.Stdout = output
		err = gofmt.Start()
		if err != nil {
			pinput.Close()
			return err
		}
		err = t.Execute(pinput, params)
		if err != nil {
			return err
		}
		err = pinput.Close()
		if err != nil {
			useGoFmt = false
			goto tryagain
		}
		err = gofmt.Wait()
		if err != nil {
			useGoFmt = false
			goto tryagain
		}
	} else {
		err = t.Execute(output, params)
	}

	return err
}

// func GenGo(b *il.ILBlock, output io.Writer) error {
// var usegofmt bool = true
// var err error

// 	_, err = exec.LookPath("gofmt")
// 	if err != nil {
// 		usegofmt = false
// 	}

// 	var body TemplateBody
// 	body.InitialDataSize = DefaultDataSize
// 	body.Body = p.CreateILTree()
// 	fmt.Fprintln(os.Stderr, "# Raw #")
// 	body.Body.Dump(os.Stderr, 0)
// 	body.Body.Optimize()
// 	fmt.Fprintln(os.Stderr, "\n# Optimized #")
// 	body.Body.Dump(os.Stderr, 0)
// 	body.Body.Prune()
// 	fmt.Fprintln(os.Stderr, "\n# Pruned #")
// 	body.Body.Dump(os.Stderr, 0)

// 	t := template.Must(template.New("body").Parse(mainfiletemplate))

// tryagain:
// 	if usegofmt {
// 		gofmt := exec.Command("gofmt")
// 		pinput, err := gofmt.StdinPipe()
// 		if err != nil {
// 			pinput.Close()
// 			usegofmt = false
// 			goto tryagain
// 		}
// 		gofmt.Stdout = output
// 		err = gofmt.Start()
// 		if err != nil {
// 			pinput.Close()
// 			return err
// 		}
// 		err = t.Execute(pinput, body)
// 		if err != nil {
// 			return err
// 		}
// 		err = pinput.Close()
// 		if err != nil {
// 			usegofmt = false
// 			goto tryagain
// 		}
// 		err = gofmt.Wait()
// 		if err != nil {
// 			usegofmt = false
// 			goto tryagain
// 		}
// 	} else {
// 		err = t.Execute(output, body)
// 	}

// return err
// }
