package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
)

var errRE = regexp.MustCompile(`(?i)error`)

func main() {
	var s *bufio.Scanner
	switch len(os.Args) {
	case 1:
		log.Println("No file specified, using stdin")
		s = bufio.NewScanner(os.Stdin)
	case 2:
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		s = bufio.NewScanner(f)
	default:
		log.Println("too many arguments provided")
		os.Exit(1)
	}

	for s.Scan() {
		line := s.Bytes()
		if errRE.Match(line) {
			fmt.Printf("%s\n", line)
		}
	}
	if err := s.Err(); err != nil {
		log.Println("Error: ", err)
		os.Exit(1)
	}
}
