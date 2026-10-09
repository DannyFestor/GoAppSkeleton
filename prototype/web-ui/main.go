// PROTOTYPE, throwaway. Answers the "Web UI prototype" wayfinder ticket:
// three structurally different web UIs over the core catalogue, switchable
// with ?variant=A|B|C on one route. Run: go run . serve --open
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintf(os.Stderr, "usage: %s serve [--addr host:port] [--open]\n", binName)
		os.Exit(2)
	}
	if err := serve(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := flags.String("addr", "127.0.0.1:7070", "address to listen on; use 0.0.0.0:7070 inside a container")
	open := flags.Bool("open", false, "open the web UI in the default browser")
	_ = flags.Parse(args)

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	url := "http://" + listener.Addr().String()
	printBanner(url)
	if *open {
		openBrowser(url)
	}
	return http.Serve(listener, routes())
}

func printBanner(url string) {
	fmt.Fprintf(os.Stderr, "%s serve %s\n", binName, generatorVersion)
	fmt.Fprintf(os.Stderr, "  web UI:  %s\n", url)
	fmt.Fprintln(os.Stderr, "  Skeletons download as zip files; nothing is written on this machine.")
	fmt.Fprintln(os.Stderr, "  Ctrl-C to stop.")
}

func openBrowser(url string) {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	_ = exec.Command(command, url).Start()
}

func routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handlePage)
	mux.HandleFunc("GET /configuration.toml", handleConfigurationFile)
	mux.HandleFunc("GET /skeleton.zip", handleZip)
	mux.HandleFunc("POST /load", handleLoad)
	return mux
}
