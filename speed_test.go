package main

import (
	"io"
	"log"
	"net"
	_ "net/http/pprof"
)

func main2() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Listening on port 8080")
	defer l.Close()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			// Shut down the connection.
			defer c.Close()

			conn2, err := net.Dial("tcp", "localhost:5201")
			if err != nil {
				log.Fatal(err)
			}

			ch := make(chan []byte)
			ch2 := make(chan []byte)
			eCh := make(chan error)

			// Start a goroutine to read from our net connection
			go func(ch chan []byte, eCh chan error) {
				for {
					// try to read the data
					data := make([]byte, 512)
					_, err := conn.Read(data)
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					ch <- data
				}
			}(ch, eCh)

			go func(ch chan []byte, eCh chan error) {
				for {
					// try to read the data
					data := make([]byte, 512)
					_, err := conn2.Read(data)
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					ch <- data
				}
			}(ch2, eCh)

			//ticker := time.Tick(time.Second)
			// continuously read from the connection
			for {
				select {
				case data := <-ch: // This case means we recieved data on the connection
					conn2.Write(data)
				case data := <-ch2: // This case means we recieved data on the connection
					c.Write(data)
				case err := <-eCh: // This case means we got an error and the goroutine has finished
					if err == io.EOF {
						log.Print("Client disconnected")
					} else {
						log.Fatalf("%s\n", err)
						// handle our error then exit for loop
						break
						// This will timeout on the read.
						//case <-ticker:
						// do nothing? this is just so we can time out if we need to.
						// you probably don't even need to have this here unless you want
						// do something specifically on the timeout.
					}
				}
			}
		}(conn)
	}
}
