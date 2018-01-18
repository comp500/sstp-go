package main

import (
	"fmt"
	"log"
	"net"
)

func handleErr(err error) {
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

func main() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
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

			// Echo all incoming data.
			//io.Copy(c, c)

			//b, err := ioutil.ReadAll(c)

			var method, path, version string
			n, err := fmt.Fscan(c, &method, &path, &version)
			handleErr(err)

			if n != 3 {
				log.Print("Malformed HTTP")
				return
			}
			if method != "SSTP_DUPLEX_POST" {
				log.Print("Wrong method.")
				return
			}
			if path != "/sra_{BA195980-CD49-458b-9E23-C84EE0ADCD75}/" {
				log.Print("Wrong path.")
				return
			}

			n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n",
				"HTTP/1.1 200 OK",
				"Date: Thu, 09 Nov 2006 00:51:09 GMT",
				"Server: Microsoft-HTTPAPI/2.0",
				"Content-Length: 18446744073709551615")
			handleErr(err)
			log.Printf("%v", n)

			/*b, err := ioutil.ReadAll(c)
			handleErr(err)
			log.Printf("%s\n", b)*/

			ch := make(chan []byte)
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
					// send data if we read some.
					ch <- data
				}
			}(ch, eCh)

			//ticker := time.Tick(time.Second)
			// continuously read from the connection
			for {
				select {
				// This case means we recieved data on the connection
				case data := <-ch:
					// Do something with the data
					// This case means we got an error and the goroutine has finished
					log.Printf("%s\n", data)
				case err := <-eCh:
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
		}(conn)
	}
}
