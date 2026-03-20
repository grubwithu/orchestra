package main

import (
	"flag"
	"log"
	"os"
	"os/exec"

	"github.com/grubwithu/hfc/internal/webcore"
)

func main() {
	// Define command line arguments
	executablePath := flag.String("program", "", "Program executable path, format: -program=xx.out")
	srcPathMatch := flag.String("srcpath", "build__HFC_qzmp__", "Replace the matched dir name in the source path, format: -srcpath=build__HFC_qzmp__")
	fuzzIntroPrefix := flag.String("fuzzintro", "fuzzerLogFile-", "Prefix of the fuzz intro file, format: -fuzzintro=fuzzerLogFile-")
	port := flag.Int("port", 8080, "Port number for the web server (default: 8080), format: -port=8080")
	help := flag.Bool("h", false, "Display help information")

	// Parse command line arguments
	flag.Parse()

	// Show help information and exit if -h is provided
	if *help || *executablePath == "" || *fuzzIntroPrefix == "" {
		log.Println("Program Usage:")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Check whether llvm-profdata and llvm-cov are installed
	if _, err := exec.LookPath("llvm-profdata"); err != nil {
		log.Fatal("Error: llvm-profdata is not installed\n")
	}
	if _, err := exec.LookPath("llvm-cov"); err != nil {
		log.Fatal("Error: llvm-cov is not installed\n")
	}

	webServer := webcore.NewServer(*port, *executablePath, *fuzzIntroPrefix, *srcPathMatch)
	webServer.Start()

	log.Println("We are done here. Have a nice day!")

}
