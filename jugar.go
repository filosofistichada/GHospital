package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"costenos-narcolepsia/presentacion"
	"costenos-narcolepsia/virus"
)

// pausaBot es lo que espera la consola entre turnos de bots para que se
// alcance a leer lo que pasó.
const pausaBot = 700 * time.Millisecond

// partida conecta el juego con la consola: lee lo que escribe la persona,
// llama al paquete virus y muestra el resultado. Las reglas viven en virus.
type partida struct {
	j       *virus.Juego
	c       *presentacion.Consola
	entrada *bufio.Scanner
}

// jugarVirus corre una partida de "Virus del Sueño" en la consola.
func jugarVirus(c *presentacion.Consola) {
	p := partida{c: c, entrada: bufio.NewScanner(os.Stdin)}
	fmt.Println(c.Titulo(`Virus del Sueño: edición "Hospital de los Costeños"`))
	fmt.Println(c.Nota("Gana el primero con 4 pacientes sanos de equipos distintos (el comodín vale por cualquiera). Ataques de sueño a los rivales, camas a los tuyos. Cuidado con el Coste."))

	nombre, ok := p.preguntar("¿Cómo te llamas? [Sofía]: ")
	if !ok {
		return
	}
	if nombre == "" {
		nombre = "Sofía"
	}
	numBots, ok := p.preguntarNumero("¿Contra cuántos bots juegas? (1-3) [2]: ", 1, 3, 2)
	if !ok {
		return
	}
	j, err := virus.NuevoJuego(nombre, numBots, rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)))
	if err != nil {
		fmt.Println(c.Error(err))
		return
	}
	p.j = j

	for j.Ganador() == nil {
		if j.Jugadores()[j.Turno()].EsBot() {
			if err := j.JugarTurnoBot(); err != nil {
				fmt.Println(c.Error(err))
				return
			}
			p.mostrarMensajes()
			time.Sleep(pausaBot)
			continue
		}
		if !p.turnoPersona() {
			fmt.Println(c.Nota("Partida abandonada. El Coste dice que él también se fue a comprar hielo."))
			return
		}
	}
	p.mostrarMesa()
	fmt.Println(c.Titulo("Fin de la partida"))
	fmt.Printf("  Ganó %s. ¡Que suene el picó!\n", j.Ganador().Nombre())
}

// turnoPersona muestra la mesa y repite hasta que la persona haga una jugada
// válida. Devuelve false si quiere salir o se acabó la entrada.
func (p *partida) turnoPersona() bool {
	p.mostrarMesa()
	yo := p.j.Jugadores()[p.j.Turno()]
	if len(yo.Mano()) == 0 {
		fmt.Println("  No tienes cartas y el mazo está vacío: pasas.")
		return p.j.Pasar() == nil
	}
	fmt.Println(p.c.Nota("Escribe: j N (jugar la carta N), d N [N N] (descartar), ? (ayuda), s (salir)."))
	for {
		linea, ok := p.preguntar("> ")
		if !ok {
			return false
		}
		campos := strings.Fields(strings.ToLower(linea))
		if len(campos) == 0 {
			continue
		}
		var err error
		switch campos[0] {
		case "s", "salir":
			return false
		case "?", "ayuda":
			p.mostrarAyuda(yo.Mano())
			continue
		case "j", "jugar":
			err = p.jugar(campos[1:], yo.Mano())
		case "d", "descartar":
			var indices []int
			indices, err = aIndices(campos[1:])
			if err == nil {
				err = p.j.Descartar(indices...)
			}
		default:
			fmt.Println("  No entendí. Escribe ? para ver la ayuda.")
			continue
		}
		if err != nil {
			fmt.Println(p.c.Error(err))
			continue
		}
		p.mostrarMensajes()
		return true
	}
}

// jugar pide el objetivo que necesita la carta elegida y la juega.
func (p *partida) jugar(args []string, mano []virus.Carta) error {
	indices, err := aIndices(args)
	if err != nil || len(indices) != 1 {
		return virus.ErrCartaInvalida
	}
	i := indices[0]
	if i < 0 || i >= len(mano) {
		return virus.ErrCartaInvalida
	}
	obj, ok := p.pedirObjetivo(mano[i].Requiere())
	if !ok {
		return fmt.Errorf("jugada cancelada")
	}
	return p.j.JugarCarta(i, obj)
}

// pedirObjetivo pregunta lo que la carta declara que necesita. Devuelve
// false si la persona cancela.
func (p *partida) pedirObjetivo(req virus.Requisitos) (virus.Objetivo, bool) {
	var obj virus.Objetivo
	jugadores := p.j.Jugadores()
	if req.Rival {
		var rivales []int
		for i := range jugadores {
			if i != p.j.Turno() {
				rivales = append(rivales, i)
			}
		}
		obj.Rival = rivales[0]
		if len(rivales) > 1 {
			for n, i := range rivales {
				fmt.Printf("  [%d] %s\n", n+1, jugadores[i].Nombre())
			}
			n, ok := p.preguntarNumero("¿A qué rival? (0 cancela): ", 0, len(rivales), -1)
			if !ok || n == 0 {
				return obj, false
			}
			obj.Rival = rivales[n-1]
		}
	}
	if req.PacienteRival {
		color, ok := p.elegirPaciente(jugadores[obj.Rival], "¿A cuál de sus pacientes? (0 cancela): ")
		if !ok {
			return obj, false
		}
		obj.PacienteRival = color
	}
	if req.PacientePropio {
		color, ok := p.elegirPaciente(jugadores[p.j.Turno()], "¿Cuál de tus pacientes? (0 cancela): ")
		if !ok {
			return obj, false
		}
		obj.PacientePropio = color
	}
	return obj, true
}

// elegirPaciente muestra los pacientes de un jugador y devuelve el color del
// elegido.
func (p *partida) elegirPaciente(jug *virus.Jugador, pregunta string) (virus.Color, bool) {
	pacientes := jug.Pacientes()
	if len(pacientes) == 0 {
		fmt.Printf("  %s no tiene pacientes en el tablero.\n", jug.Nombre())
		return 0, false
	}
	fmt.Println(presentacion.ListaPacientesVirus(jug))
	n, ok := p.preguntarNumero(pregunta, 0, len(pacientes), -1)
	if !ok || n == 0 {
		return 0, false
	}
	return pacientes[n-1].Color(), true
}

// mostrarMesa dibuja los tableros de los rivales arriba y el propio abajo,
// con la mano de la persona.
func (p *partida) mostrarMesa() {
	jugadores := p.j.Jugadores()
	fmt.Println(p.c.Titulo(fmt.Sprintf("La mesa (quedan %d cartas en el mazo)", p.j.CartasEnMazo())))
	for i := 1; i < len(jugadores); i++ {
		fmt.Println(p.c.TableroVirus(jugadores[i], i, p.j.Turno() == i))
	}
	fmt.Println()
	fmt.Println(p.c.TableroVirus(jugadores[0], 0, p.j.Turno() == 0))
	if !jugadores[0].EsBot() && p.j.Ganador() == nil {
		fmt.Println("\n  Tu mano:")
		fmt.Println(p.c.ManoVirus(jugadores[0].Mano()))
	}
}

// mostrarAyuda explica los comandos y las cartas de la mano.
func (p *partida) mostrarAyuda(mano []virus.Carta) {
	fmt.Println("  j N        juega la carta N; te pregunta a quién si hace falta")
	fmt.Println("  d N [N N]  descarta de 1 a 3 cartas y robas otras")
	fmt.Println("  s          sale de la partida")
	for i, c := range mano {
		fmt.Printf("  [%d] %s: %s\n", i+1, c.Nombre(), c.Descripcion())
	}
}

// mostrarMensajes imprime los chistes y frases que dejó la última jugada.
func (p *partida) mostrarMensajes() {
	for _, m := range p.j.TomarMensajes() {
		fmt.Println(p.c.Nota("  " + m))
	}
}

// preguntar lee una línea. Devuelve false si se acabó la entrada.
func (p *partida) preguntar(pregunta string) (string, bool) {
	fmt.Print(pregunta)
	if !p.entrada.Scan() {
		fmt.Println()
		return "", false
	}
	return strings.TrimSpace(p.entrada.Text()), true
}

// preguntarNumero lee un número entre min y max; con la línea vacía devuelve
// porDefecto si está en el rango. Repite hasta que la respuesta sea válida.
func (p *partida) preguntarNumero(pregunta string, min, max, porDefecto int) (int, bool) {
	for {
		linea, ok := p.preguntar(pregunta)
		if !ok {
			return 0, false
		}
		if linea == "" && porDefecto >= min && porDefecto <= max {
			return porDefecto, true
		}
		if n, err := strconv.Atoi(linea); err == nil && n >= min && n <= max {
			return n, true
		}
		fmt.Printf("  Escribe un número de %d a %d.\n", min, max)
	}
}

// aIndices convierte "1 3" (como lo ve la persona) en índices 0 y 2.
func aIndices(args []string) ([]int, error) {
	indices := make([]int, 0, len(args))
	for _, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil {
			return nil, virus.ErrCartaInvalida
		}
		indices = append(indices, n-1)
	}
	return indices, nil
}
