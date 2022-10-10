package main

import (
	"flag"
	"fmt"
	"io"
	"log"

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

		opt := b2b.DefaultOptions()
		// opt.MaxConnectionsPerPeer = 1

		b, _ := b2b.New(a, &opt)
		b.AddProcol("test-protocol", func(s *b2b.Stream) {
			defer s.Close()
			b, _ := io.ReadAll(s)
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

		b, _ := b2b.New(a, nil)

		peerID, err := b.Connect("localhost:" + *port)
		if err != nil {
			log.Fatal(err)
		}
		s, err := b.NewStream("test-protocol", peerID)
		if err != nil {
			log.Fatal(err)
		}

		_, err = s.Write([]byte("yo yo yo yo"))
		if err != nil {
			log.Fatal(err)
		}

		m, err := io.ReadAll(s)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		m, err = io.ReadAll(s)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		m, err = io.ReadAll(s)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(m))

		s.Close()
		b.Disconnect(peerID)
	}
}
