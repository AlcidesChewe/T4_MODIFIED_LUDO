package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

const ServerAddress = "localhost:8080"

const (
	BoardSize = 40
	Finish    = BoardSize + 1
	Obstacle  = -2
)

type Player struct {
	ID         int
	characters []int
	conn       net.Conn
	board      []int
}

func (p *Player) Play() {
	reader := bufio.NewReader(p.conn)
	for {
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)
		if msg == "Game Over" {
			fmt.Println(msg)
			break
		}

		if msg == "Your turn" {
			// Lógica para decidir movimiento
			move := p.decideMove()
			fmt.Fprintln(p.conn, strconv.Itoa(move))
		}
	}
}

func (p *Player) decideMove() int {
	dice1 := rand.Intn(6) + 1
	dice2 := rand.Intn(6) + 1
	operation := rand.Intn(2) // 0 para sumar, 1 para restar

	move := dice1
	if operation == 0 {
		move += dice2
	} else {
		move -= dice2
	}

	fmt.Printf("Jugador %d lanza los dados: %d y %d. Operación: %s. Movimiento: %d\n",
		p.ID, dice1, dice2, operationString(operation), move)

	// Decidir qué peón mover
	selectedPawn, _ := p.selectPawn(move)
	fmt.Printf("Jugador %d mueve peón %d\n", p.ID, selectedPawn)

	return selectedPawn
}

func operationString(operation int) string {
	if operation == 0 {
		return "+"
	}
	return "-"
}

func (p *Player) selectPawn(move int) (int, int) {
	// bestPawn será el índice del peón seleccionado para mover.
	// bestDistance será la distancia más corta al final encontrada hasta ahora.
	bestPawn := -1
	bestDistance := -1

	// Iterar sobre cada peón para determinar cuál es el mejor para mover.
	for i, pos := range p.characters {
		// Si el peón ya ha terminado, lo ignoramos en esta ronda.
		if pos == Finish {
			continue
		}

		// Calcular la nueva posición potencial del peón después del movimiento.
		newPos := pos + move
		// Si el nuevo movimiento excede el tamaño del tablero, lo ajustamos al límite.
		if newPos > BoardSize {
			newPos = BoardSize
		}

		// Si la nueva posición tiene un obstáculo, ignoramos este peón para este turno.
		if p.board[newPos] == Obstacle {
			continue
		}

		// Calculamos la distancia desde la nueva posición hasta la meta.
		distanceToFinish := BoardSize - newPos
		// Si no hemos seleccionado un peón aún o si este peón está más cerca de la meta
		// que los anteriores, lo seleccionamos.
		if bestPawn == -1 || distanceToFinish < bestDistance {
			bestPawn = i
			bestDistance = distanceToFinish
		}
	}

	// Devolvemos el índice del mejor peón y su nueva posición.
	// Si no se pudo seleccionar un peón (todos bloqueados o en la meta),
	// bestPawn será -1, lo cual debe manejarse en la lógica del juego.
	return bestPawn, p.characters[bestPawn] + move
}

func main() {
	rand.Seed(time.Now().UnixNano())

	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Println("Error al conectar con el servidor:", err)
		return
	}
	defer conn.Close()

	player := &Player{
		conn: conn,
	}
	player.Play()
}