package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

func handleErr(err error) {
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

type parseReturn struct {
	isControl bool
	Data      []byte
}

func main() {
	runtime.SetBlockProfileRate(1)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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

			var method, path, version string
			n, err := fmt.Fscan(c, &method, &path, &version)
			handleErr(err)

			if n != 3 {
				log.Print("Malformed HTTP")
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 400 Bad Request",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 15",
					"400 Bad Request")
				handleErr(err)
				log.Printf("%v HTTP bytes written (400)", n)
				return
			}
			if method != "SSTP_DUPLEX_POST" {
				log.Printf("Wrong method (%s)", method)
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 405 Method Not Allowed",
					"Allow: SSTP_DUPLEX_POST",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 22",
					"405 Method Not Allowed")
				handleErr(err)
				log.Printf("%v HTTP bytes written (405)", n)
				return
			}
			if path != "/sra_{BA195980-CD49-458b-9E23-C84EE0ADCD75}/" {
				log.Printf("Wrong path (%s)", path)
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 404 File Not Found",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 18",
					"404 File Not Found")
				handleErr(err)
				log.Printf("%v HTTP bytes written (404)", n)
				return
			}

			// digest rest of first packet
			data := make([]byte, 2048)
			conn.Read(data)
			data = nil // free memory

			log.Print("HTTP request received")

			n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n",
				"HTTP/1.1 200 OK",
				"Date: Thu, 09 Nov 2006 00:51:09 GMT",
				"Server: Microsoft-HTTPAPI/2.0",
				"Content-Length: 18446744073709551615")
			handleErr(err)
			log.Printf("%v HTTP bytes written", n)

			ch := make(chan parseReturn)
			eCh := make(chan error)

			packChan := make(chan []byte)
			// TODO: shouldn't this be c, rather than conn??????
			pppdInstance := pppdInstance{nil, nil, newUnescaper(packetHandler{conn, packChan})} // store null pointer to future pppd instance

			// Start a goroutine to read from our net connection
			go func(ch chan parseReturn, eCh chan error) {
				for {
					// try to read the data
					var data [4]byte
					// TODO: shouldn't this be c, rather than conn??????
					n, err := conn.Read(data[:])
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					if n < 4 {
						eCh <- errors.New("Invalid packet")
						return
					}
					isControl, lengthToRead, err := decodeHeader(data[:])
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					newData := make([]byte, lengthToRead)
					n, err = conn.Read(newData)
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					if n != lengthToRead {
						eCh <- errors.New("Not all of packet read")
						return
					}
					ch <- parseReturn{isControl, newData}
				}
			}(ch, eCh)

			go func(packChan chan []byte) {
				for {
					select {
					case data := <-packChan: // This case means we recieved data on the connection
						conn.Write(data)
					}
				}
			}(packChan)

			//ticker := time.Tick(time.Second)
			// continuously read from the connection
			for {
				select {
				case data := <-ch: // This case means we recieved data on the connection
					// Do something with the data
					//log.Printf("%s\n", hex.Dump(data))
					if data.isControl {
						header := parseControl(data.Data)
						handleControlPacket(header, conn, &pppdInstance)
					} else {
						handleDataPacket(data.Data, conn, &pppdInstance)
					}
				case err := <-eCh: // This case means we got an error and the goroutine has finished
					if err == io.EOF {
						log.Print("Client disconnected")
						if pppdInstance.commandInst != nil {
							// kill pppd if disconnect
							err := pppdInstance.commandInst.Process.Kill()
							handleErr(err)
							pppdInstance.commandInst = nil
						}
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
