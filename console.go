package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Console() {

	for {
		fmt.Print("Enter text: ")

		bio := bufio.NewReader(os.Stdin)
		line, _ := bio.ReadString('\n')

		switch {
		case strings.HasPrefix(line, "pipe"):
			pipe(line)
		case strings.HasPrefix(line, "point"):
			point(line)
		case strings.HasPrefix(line, "join"):
			joinchange(line)

		}
		fmt.Println(line)
	}
}

func pipe(line string) {
	var pipeid int
	fmt.Sscanf(line, "pipe %d", &pipeid)
	fmt.Println(pipeid)
}

func point(line string) {
	var point int
	fmt.Sscanf(line, "point %d", &point)
	fmt.Println(point)
	path := t.FindPathID(point)
	fmt.Println(path)
	fmt.Println(t.Connections[point])
}

func joinchange(line string) {
	var j int
	fmt.Sscanf(line, "join %d", &j)
	t.JoinRadius = j
	fmt.Println("join length now ", j, " meters")
}
