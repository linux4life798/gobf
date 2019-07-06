//go:generate ./gen_template_consts.bash
package lang

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"

	"github.com/linux4life798/gobf/gobflib/il"
)

const (
	DefaultDataSize = 100000
)

type TemplateParams struct {
	InitialDataSize  int
	Body             <-chan string
	ProfilingEnabled bool
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
		cout <- "for data[datap] != 0 {"
		for _, ib := range b.GetInner() {
			ilBlockGo(ib, cout)
		}
		cout <- "}"
	case il.ILDataPtrAdd:
		cout <- fmt.Sprintf("datapadd(%d)", b.GetParam())
	case il.ILDataAdd:
		cout <- fmt.Sprintf("dataadd(%v)", byte(b.GetParam()))
	case il.ILRead:
		for i := int64(0); i < b.GetParam(); i++ {
			cout <- "readb()"
		}
	case il.ILWrite:
		cout <- fmt.Sprintf("writeb(%v)", b.GetParam())
	case il.ILDataAddVector:
		cout <- fmt.Sprintf("dataaddvector(%#v)", b.GetVector())
	case il.ILDataSet:
		cout <- fmt.Sprintf("dataset(%d)", byte(b.GetParam()))
	default:
		panic("Encountered an unknown ILBlock type.")
	}
}

func ILBlockToGo(b *il.ILBlock, output io.Writer, profileenabled bool) error {
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
		InitialDataSize:  DefaultDataSize,
		Body:             c,
		ProfilingEnabled: profileenabled,
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

func CompileGo(infile, outfile string, debugenabled bool, gccgo bool) error {
	var args = []string{"build"}
	if gccgo {
		args = append(args, "-compiler", "gccgo")
	}
	args = append(args, "-o", outfile, infile)

	gobuild := exec.Command("go", args...)
	gobuild.Stdout = os.Stderr
	gobuild.Stderr = os.Stderr
	if err := gobuild.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary from Go: %v\n", err)
		return err
	}
	if err := gobuild.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary from Go: %v\n", err)
		return err
	}
	return nil
}

// If err is non-nil, the tempdir is preserved and returned
// with the error
func CompileIL(b *il.ILBlock, outfile string, debugenabled, profileenabled bool) (error, string) {
	// Create temp directory for generated Go /tmp/gobfcompile########
	tempdir, err := ioutil.TempDir("", "gobfcompile")
	if err != nil {
		return fmt.Errorf("Failed to create temp dir: %v", err), tempdir
	}

	// Create temp /tmp/gobfcompile########/main.go file
	gofile, err := os.Create(tempdir + "/main.go")
	if err != nil {
		return fmt.Errorf("Failed to create temp file: %v", err), tempdir
	}

	// Generate the Go code
	if err := ILBlockToGo(b, gofile, profileenabled); err != nil {
		return fmt.Errorf("Failed to generate Go: %v", err), tempdir
	}

	// Compile the Go code to binary
	if err := CompileGo(gofile.Name(), outfile, debugenabled, false); err != nil {
		return fmt.Errorf("Failed to compile generated Go: %v", err), tempdir
	}

	if !debugenabled {
		// Remove the temp directory if everything succeeded
		if err := os.RemoveAll(tempdir); err != nil {
			return err, tempdir
		}
	}

	return nil, tempdir
}
