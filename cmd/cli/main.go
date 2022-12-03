package main

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"os"
)

func main() {
	arg1, arg2, arg3, err := validateInput()
	//validate if we hae any argument with command
	if err != nil {
		exitGracefully(err)
	}
	switch arg1 {
	case "help":
		showhelp()
	case "version":
		fmt.Println("1.0.0")
	case "new":
		initiateApp()

	default:
		showhelp()
	}
	fmt.Println(arg1, arg2, arg3)
}

func validateInput() (string, string, string, error) {
	var arg1, arg2, arg3 string
	if len(os.Args) > 1 {

		arg1 = os.Args[1]
		if len(os.Args) >= 3 {
			arg2 = os.Args[2]
		}
		if len(os.Args) >= 4 {
			arg3 = os.Args[3]
		}
		if len(os.Args) > 4 {
			color.Red("Too many arguments")
		}

	} else {
		color.Red("No argument provided, Please provide at least one argument")
		showhelp()
		return "", "", "", errors.New("command required")
	}

	return arg1, arg2, arg3, nil

}

func exitGracefully(err error, msg ...string) {
	message := ""
	if len(msg) > 0 {
		message = msg[0]
	}
	if err != nil {
		color.Red("Error:%v\n", err)
	}
	if len(message) > 0 {
		color.Yellow(message)
	} else {
		color.Green("Finished")
	}
	os.Exit(0)

}
func showhelp() {
	color.Yellow(`
	Available Commands :
	help  - commands
	version - show version of the application
	

	`)

}
