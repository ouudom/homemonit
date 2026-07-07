package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/agent"
	"github.com/henrygd/beszel/agent/health"
	"github.com/spf13/pflag"
)

type cmdOptions struct {
	hubURL string
	token  string
}

func (opts *cmdOptions) parse() bool {
	subcommand := ""
	if len(os.Args) > 1 {
		subcommand = os.Args[1]
	}

	switch subcommand {
	case "health":
		err := health.Check()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print("ok")
		return true
	}

	pflag.StringVarP(&opts.hubURL, "url", "u", "", "URL of the HomeMonit hub")
	pflag.StringVarP(&opts.token, "token", "t", "", "Token to use for authentication")
	version := pflag.BoolP("version", "v", false, "Show version information")
	help := pflag.BoolP("help", "h", false, "Show this help message")

	// Convert old single-dash long flags to double-dash
	flagsToConvert := []string{"url", "token"}
	for i, arg := range os.Args {
		for _, flag := range flagsToConvert {
			singleDash := "-" + flag
			doubleDash := "--" + flag
			if arg == singleDash {
				os.Args[i] = doubleDash
				break
			} else if strings.HasPrefix(arg, singleDash+"=") {
				os.Args[i] = doubleDash + arg[len(singleDash):]
				break
			}
		}
	}

	pflag.Usage = func() {
		builder := strings.Builder{}
		builder.WriteString("Usage: ")
		builder.WriteString(os.Args[0])
		builder.WriteString(" [command] [flags]\n")
		builder.WriteString("\nCommands:\n")
		builder.WriteString("  health       Check if the agent is running\n")
		builder.WriteString("\nFlags:\n")
		fmt.Print(builder.String())
		pflag.PrintDefaults()
	}

	pflag.Parse()

	switch {
	case *version:
		fmt.Println(beszel.AppName+"-agent", beszel.Version)
		return true
	case *help || subcommand == "help":
		pflag.Usage()
		return true
	}

	if opts.hubURL != "" {
		os.Setenv("HUB_URL", opts.hubURL)
	}
	if opts.token != "" {
		os.Setenv("TOKEN", opts.token)
	}
	return false
}

func main() {
	var opts cmdOptions
	subcommandHandled := opts.parse()

	if subcommandHandled {
		return
	}

	a, err := agent.NewAgent()
	if err != nil {
		log.Fatal("Failed to create agent: ", err)
	}

	var serverConfig agent.AgentOptions
	if err := a.Start(serverConfig); err != nil {
		log.Fatal("Failed to start: ", err)
	}
}
