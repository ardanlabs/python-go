package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		name := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "usage: %s [FILE]", name)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 2 {
		log.Fatal("error: wrong number of arguments")
	}

	var r io.Reader
	if flag.NArg() == 1 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatalf("error: %s", err)
		}
		defer file.Close()
		r = file
	} else {
		r = os.Stdin
	}

	s := bufio.NewScanner(r)
	lnum := 0
	for s.Scan() {
		lnum++
		line := strings.ReplaceAll(s.Text(), "\t", "    ")
		fmt.Printf("% 3d %s\n", lnum, line)
	}

	if err := s.Err(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
