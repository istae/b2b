package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/istae/b2b"
)

func main() {

	isServer := flag.Bool("server", false, "")
	port := flag.String("port", "", "")

	flag.Parse()

	if *isServer {

		a, err := b2b.NewAsymmetric()
		if err != nil {
			log.Fatal(err)
		}

		b, _ := b2b.NewB2B(a)
		b.AddProcol("test-protocol", func(s *b2b.Stream) {
			defer s.Close()
			b, _ := s.Read()
			fmt.Println("handle: test-protocol", string(b))
			s.Write([]byte("what up what up"))
			s.Write([]byte("what up what up"))
			s.Write([]byte("what up what up"))
		})

		fmt.Println("startig server")
		err = b.Listen(fmt.Sprintf("localhost:%s", *port))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		a, err := b2b.NewAsymmetric()
		if err != nil {
			log.Fatal(err)
		}

		b, _ := b2b.NewB2B(a)
		b.SetStreamMaxInactive(time.Second)

		peerID, err := b.Connect("localhost:" + *port)
		if err != nil {
			log.Fatal(err)
		}
		defer b.Disconnect(peerID)
		s, err := b.NewStream("test-protocol", peerID)
		if err != nil {
			log.Fatal(err)
		}
		defer s.Close()

		err = s.Write([]byte("yo yo yo yo"))
		if err != nil {
			log.Fatal(err)
		}

		m, err := s.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		m, err = s.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		m, err = s.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		b.Disconnect(peerID)
	}
}
