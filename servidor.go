package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BoardSize     = 64
	NumPlayers    = 4
	NumCharacters = 4
	NumObstacles  = 10
	ServerAddress = "localhost:8080"
)

type Player struct {
	ID         uint
	characters []int
	missTurn   chan bool
	conn       net.Conn
	buff       *bufio.Reader
}

type BoardSquare uint8

const (
	PATH     BoardSquare = 0
	WALL     BoardSquare = 1
	TRAP     BoardSquare = 2
	CREATURE BoardSquare = 3
)

type Game struct {
	board      []BoardSquare
	gameOver   bool
	winner     uint
	turnSignal chan int
	players    []*Player
}

func (p *Player) Play(g *Game) {
	for true {
		_, ok := <-g.turnSignal
		if !ok {
			fmt.Fprintln(p.conn, "Ganó el jugador "+strconv.Itoa(int(g.winner)))
			return
		}
		fmt.Printf("Turno del jugador %d\n", p.ID)

		if <-p.missTurn {
			g.turnSignal <- 1
			p.missTurn <- false
			fmt.Printf("El jugador %d ha perdido su turno\n", p.ID)
			fmt.Fprintln(p.conn, "miss")
			continue
		}

		// Enviar señal de jugada
		fmt.Fprintln(p.conn, "play")

		// Recibir jugada del jugadador
		str, _ := p.buff.ReadString('\n')
		datos := strings.Split(str, " ")
		switch datos[0] {
		case "move":
			idx, _ := strconv.Atoi(datos[1])
			pos, _ := strconv.Atoi(datos[2])
			p.characters[idx] = pos
			p.missTurn <- false
		case "miss":
			p.missTurn <- true
		}

		time.Sleep(500 * time.Millisecond)

		if isWinner(p.characters) {
			fmt.Printf("El jugador %d ganó la partida.\nFIN DEL JUEGO.\n", p.ID)
			g.winner = p.ID
			g.gameOver = true
			close(g.turnSignal)
			fmt.Fprintln(p.conn, "win")
			return
		}

		if !g.gameOver {
			g.turnSignal <- 1
		}
	}
}

func isWinner(positions []int) bool {
	for _, pos := range positions {
		if pos < BoardSize {
			return false
		}
	}
	return true
}

func PrintBoard(board []BoardSquare, players []*Player) {
	width := int(math.Sqrt(BoardSize))
	var occupied BoardSquare = 9
	for _, p := range players {
		for _, c := range p.characters {
			if c < BoardSize {
				board[c] = occupied
			}
		}
	}
	for i := 0; i < BoardSize; i++ {
		fmt.Printf("%d ", board[i])
		if (i+1)%width == 0 {
			fmt.Printf("\n")
		}
	}
}

func main() {
	game := &Game{
		board:      make([]BoardSquare, BoardSize),
		turnSignal: make(chan int, 1),
	}

	for i := 0; i < NumObstacles; i++ {
		min := len(game.board) / NumObstacles * i
		max := min + NumObstacles
		obsPos := rand.Intn(max-min) + min
		game.board[obsPos] = BoardSquare(rand.Intn(3) + 1)
	}
	game.turnSignal <- 1

	// Codificar el tablero para pasarlo como referencia a cada jugador
	boardRefBytes, _ := json.Marshal(game.board)
	boardRefStr := string(boardRefBytes)

	PrintBoard(game.board, game.players)
	game.players = make([]*Player, NumPlayers)

	listener, err := net.Listen("tcp", ServerAddress)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("Esperando a que se conecten todos los jugadores...")

	for i := 0; i < NumPlayers; i++ {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error aceptando la conexión al servidor: %v\n", err)
			return
		}
		buff := bufio.NewReader(conn)
		game.players[i] = &Player{
			ID:         uint(i + 1),
			characters: make([]int, NumCharacters),
			missTurn:   make(chan bool, 1),
			conn:       conn,
			buff:       buff,
		}
		// Inicializar la pérdida del turno en falso
		game.players[i].missTurn <- false

		fmt.Fprintln(conn, strconv.Itoa(i+1)+" "+strconv.Itoa(NumCharacters)+" "+boardRefStr)
		fmt.Fprintln(conn, "Conectado: Eres el jugador "+strconv.Itoa(i+1))
	}

	fmt.Println("Todos los jugadores se han conectado. Esperando a que comience al juego...")
	fmt.Println("INICIO DEL JUEGO")

	var wg sync.WaitGroup

	for _, player := range game.players {
		wg.Add(1)
		go func(p *Player) {
			defer wg.Done()
			p.Play(game)
		}(player)
		fmt.Fprintln(player.conn, "INICIO DEL JUEGO")
	}

	wg.Wait()
	PrintBoard(game.board, game.players)
}
