package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	udpaddr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		log.Fatalf("unable to resolve udp address: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, udpaddr)
	if err != nil {
		log.Fatalf("unable to dial udp: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">")
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("unable to read string: %v", err)
		}
		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Fatalf("unable to write line to connection: %v", err)
		}
	}
}
