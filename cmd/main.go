package main

import (
	"flag"
	"fmt"

	"github.com/istae/b2b"
)

func main() {

	isServer := flag.Bool("server", false, "")
	port := flag.String("port", "", "")

	flag.Parse()

	b2b := b2b.New("localhost", *port)

	if *isServer {
		fmt.Println("startig server")
		err := b2b.Listen()
		if err != nil {
			fmt.Println(err)
		}
	} else {
		fmt.Println("connecting")
		_, err := b2b.Connect("localhost", *port)
		if err != nil {
			fmt.Println(err)
		}
	}
}
