package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
)

const (
	ServerAddress = "localhost:8080"
)

type LocalPlayer struct {
	ID         uint
	characters []int
	boardRef   []uint8
	boardSize  uint
	missTurn   bool
	gameOver   bool
	conn       net.Conn
}
type Best struct {
	index    int
	position int
}

func NextMovement(p *LocalPlayer, board []uint8, move int) Best {
	var wg sync.WaitGroup
	bestChann := make(chan Best, len(p.characters)+1)
	bestChann <- Best{-1, -12}

	for i, posChar := range p.characters {
		if posChar == int(p.boardSize) {
			continue
		}
		wg.Add(1)
		go func(index, position int) {
			defer wg.Done()
			newPos := position + move
			fmt.Println("\tPersonaje", index+1, "avanzaría de", position, "a", newPos)
			// Se dirige a un camino libre
			if newPos > 0 && newPos < int(p.boardSize) && board[newPos] == 0 {
				best := <-bestChann
				// Ha avanzado más
				if newPos > best.position {
					fmt.Println("\tPersonaje", index+1, "en posición", newPos, "es mejor que el personaje", best.index+1)
					current := Best{index, newPos}
					bestChann <- current
				} else {
					bestChann <- best
				}
			}
		}(i, posChar)
	}
	wg.Wait()

	best := <-bestChann
	close(bestChann)
	// Elegir aleatorio si no pudo encontrar uno ideal
	if best.index == -1 {
		best.index = rand.Intn(len(p.characters))
		best.position = p.characters[best.index] + move
	}
	fmt.Println("\tEs mejor mover el personaje", best.index+1)
	return best
}

func (p *LocalPlayer) PlayTurn() {
	dice1 := rand.Intn(6) + 1
	dice2 := rand.Intn(6) + 1
	var move int = 0

	if operator := rand.Intn(2); operator == 0 {
		move = dice1 + dice2
		fmt.Printf("Dado: %d + %d\n", dice1, dice2)
	} else {
		move = dice1 - dice2
		fmt.Printf("Dado: %d - %d\n", dice1, dice2)
	}

	best := NextMovement(p, p.boardRef, move)
	charIndex := best.index
	newPos := best.position

	if newPos < 0 {
		newPos = 0
		p.characters[charIndex] = 0
		p.missTurn = false
		fmt.Printf("Personaje %d del jugador %d regresa al inicio\n", charIndex+1, p.ID)
	} else if newPos >= int(p.boardSize) {
		newPos = int(p.boardSize)
		p.characters[charIndex] = int(p.boardSize)
		p.missTurn = false
		fmt.Printf("El personaje %d del jugador %d llegó a la meta\n", charIndex+1, p.ID)
	} else if p.boardRef[newPos] != 0 {
		// p.characters[charIndex] = newPos
		p.missTurn = true
		fmt.Printf("El personaje %d del jugador %d cayó en un obstáculo, pierde el turno\n", charIndex+1, p.ID)
	} else {
		p.characters[charIndex] = newPos
		p.missTurn = false
		fmt.Printf("El jugador %d avanzó/retrocedió el personaje %d a la casilla %d\n", p.ID, charIndex+1, p.characters[charIndex])
	}

	if !p.missTurn {
		fmt.Fprintln(p.conn, "move "+strconv.Itoa(charIndex)+" "+strconv.Itoa(newPos)+" -")
	} else {
		fmt.Fprintln(p.conn, "miss -")
	}

}

func main() {
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Printf("Error de conexión al servidor: %v\n", err)
		return
	}
	defer conn.Close()

	var boardRef []uint8
	br := bufio.NewReader(conn)
	str, _ := br.ReadString('\n')
	parts := strings.Split(str, " ")
	numPlayer, _ := strconv.Atoi(parts[0])
	numCharac, _ := strconv.Atoi(parts[1])
	json.Unmarshal([]byte(parts[2]), &boardRef)
	bSize := len(boardRef)

	localPlayer := &LocalPlayer{
		ID:         uint(numPlayer),
		characters: make([]int, numCharac),
		boardRef:   boardRef,
		boardSize:  uint(bSize),
		missTurn:   false,
		gameOver:   false,
		conn:       conn,
	}

	msg, _ := br.ReadString('\n')
	fmt.Printf(msg)
	fmt.Printf("Soy el jugador %d...\n", localPlayer.ID)
	fmt.Println("Esperando a que el juego comience...\n")

	// Esperar a que el servidor mande la señal para que inicie el juego.
	msg, _ = br.ReadString('\n')
	fmt.Println(msg)

	for !localPlayer.gameOver {
		// Esperar a que el servidor le de señal de turno
		msg, _ = br.ReadString('\n')
		msg = strings.TrimSpace(msg)

		switch msg {
		case "play":
			localPlayer.PlayTurn()
		case "miss":
			fmt.Println("Perdí mi turno...")
		case "win":
			fmt.Println("GANE!!\nFIN DEL JUEGO")
			localPlayer.gameOver = true
		default:
			fmt.Println(msg + "\nFIN DEL JUEGO")
			localPlayer.gameOver = true
		}
	}
}
