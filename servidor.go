package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BoardSize     = 40
	NumPlayers    = 4
	NumCharacters = 4
	Start         = -1
	Finish        = BoardSize + 1
	ServerAddress = "localhost:8080"
	Obstacle      = -2
)

type Player struct {
	ID         int
	characters []int
	conn       net.Conn
	buff       *bufio.Reader
}

type Game struct {
	board      []int
	gameOver   bool
	winner     int
	turnSignal chan int
	players    []*Player
	mutex      sync.Mutex
}

func NewGame() *Game {
	g := &Game{
		board:      make([]int, BoardSize),
		turnSignal: make(chan int, 1),
		players:    make([]*Player, NumPlayers),
	}
	for i := 0; i < BoardSize; i++ {
		if rand.Float32() < 0.1 { // Aprox. 10% de casillas con obstáculos
			g.board[i] = -2 // Representa un obstáculo
		}
	}
	return g
}

func (p *Player) Play(g *Game, wg *sync.WaitGroup) {
	defer wg.Done()
	for !g.gameOver {
		<-g.turnSignal
		if g.gameOver {
			break
		}

		fmt.Println("Turno del jugador", p.ID)
		fmt.Fprintln(p.conn, "Your turn")

		// Recibir movimiento del jugador
		moveStr, _ := p.buff.ReadString('\n')
		moveStr = strings.TrimSpace(moveStr)
		move, _ := strconv.Atoi(moveStr)

		g.mutex.Lock()
		// Actualizar la posición del personaje basado en el movimiento
		for i, pos := range p.characters {
			if pos != Finish && pos != Start {
				newPos := pos + move
				if newPos > BoardSize {
					p.characters[i] = Finish
				} else {
					p.characters[i] = newPos
				}
			}
		}
		g.mutex.Unlock()

		// Verificar si el jugador ha ganado
		if hasWon(p.characters) {
			g.winner = p.ID
			g.gameOver = true
		}

		if !g.gameOver {
			g.turnSignal <- 1
		}
	}
	fmt.Fprintln(p.conn, "Game Over. Ganó el jugador "+strconv.Itoa(g.winner))
}

func hasWon(positions []int) bool {
	for _, pos := range positions {
		if pos != Finish {
			return false
		}
	}
	return true
}

func main() {
	rand.Seed(time.Now().UnixNano())

	game := NewGame()
	game.turnSignal <- 1 // Comenzar con el primer jugador

	listener, err := net.Listen("tcp", ServerAddress)
	if err != nil {
		fmt.Println("Error al iniciar el servidor:", err)
		return
	}
	defer listener.Close()

	var wg sync.WaitGroup

	for i := 0; i < NumPlayers; i++ {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			return
		}
		game.players[i] = &Player{
			ID:         i,
			characters: make([]int, NumCharacters),
			conn:       conn,
			buff:       bufio.NewReader(conn),
		}
		wg.Add(1)
		go game.players[i].Play(game, &wg)
	}

	wg.Wait() // Esperar a que todos los jugadores terminen
}
