package platform

import (
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"fmt"
)

type stdoutPrinter struct {

}
type noPrinter struct {

}

func (n noPrinter) Print(v ...interface{}) {}

func (n noPrinter) Printf(format string, v ...interface{}) {}

func (n noPrinter) Println(v ...interface{}) {}

func (n noPrinter) WritesToConsole() bool { return false }

type noTerminalPrinter struct {

}

func (t noTerminalPrinter) Print(a ...interface{}) (n int, err error) {
	return 0, nil
}

func (t noTerminalPrinter) Printf(format string, a ...interface{}) (n int, err error) {
	return 0, nil
}

func (t noTerminalPrinter) Println(a ...interface{}) (n int, err error) {
	return 0, nil
}

func (n stdoutPrinter) Print(v ...interface{}) {
	fmt.Print(v...)
}

func (n stdoutPrinter) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (n stdoutPrinter) Println(v ...interface{}) {
	fmt.Println()
}

func (n stdoutPrinter) WritesToConsole() bool {
	return true
}

var _ trace.Printer = stdoutPrinter{}
var _ trace.Printer = noPrinter{}
var _ terminal.Printer = noTerminalPrinter{}