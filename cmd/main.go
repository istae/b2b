package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/istae/b2b"
)

func main() {

	isServer := flag.Bool("server", false, "")
	port := flag.String("port", "", "")

	flag.Parse()

	if *isServer {

		b := b2b.New("localhost", *port)
		b.AddProcol("test-protocol", func(s *b2b.Stream) {
			defer s.Close()
			b, _ := s.Read()
			fmt.Println("handle: test-protocol", string(b))
			s.Write([]byte("what up what up"))
		})

		fmt.Println("startig server")
		err := b.Listen()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		b := b2b.New("", "")
		peerID, err := b.Connect("localhost:" + *port)
		if err != nil {
			log.Fatal(err)
		}
		s, err := b.NewStream("test-protocol", peerID)
		if err != nil {
			log.Fatal(err)
		}

		err = s.Write([]byte("yo yo yo yo"))
		if err != nil {
			log.Fatal(err)
		}

		m, err := s.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		// s.Close()
	}
}
