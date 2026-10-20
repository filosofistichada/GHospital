package presentacion

import (
	"fmt"
	"strings"

	"costenos-narcolepsia/virus"
)

// anchoMini es el ancho del texto de las cartas del mini-juego.
const anchoMini = 20

// ansiColorJuego da el color de consola de cada color del juego.
var ansiColorJuego = map[virus.Color]string{
	virus.Rojo:       "\033[91m",
	virus.Azul:       "\033[94m",
	virus.Verde:      "\033[92m",
	virus.Amarillo:   "\033[93m",
	virus.Multicolor: "\033[95m",
}

// ManoVirus dibuja las cartas de la mano numeradas desde 1.
func (c *Consola) ManoVirus(mano []virus.Carta) string {
	cartas := make([]string, 0, len(mano))
	for i, carta := range mano {
		lineas := []string{izqDer(fmt.Sprintf("[%d]", i+1), strings.ToUpper(carta.Tipo().String()), anchoMini)}
		lineas = append(lineas, envolver(carta.Nombre(), anchoMini)...)
		lineas = append(lineas, carta.Color().String())
		lineas = append(lineas, envolver(carta.Descripcion(), anchoMini)...)
		cartas = append(cartas, c.cartaDeAncho(lineas, ansiColorJuego[carta.Color()], anchoMini))
	}
	return unirLado(cartas, anchoMini)
}

// TableroVirus dibuja el tablero de un jugador con un encabezado. numero es
// el que se usa para elegirlo como rival (0 si no aplica).
func (c *Consola) TableroVirus(jug *virus.Jugador, numero int, deTurno bool) string {
	encabezado := jug.Nombre()
	if numero > 0 {
		encabezado = fmt.Sprintf("[%d] %s", numero, encabezado)
	}
	if jug.EsBot() {
		encabezado += " (bot)"
	}
	encabezado += fmt.Sprintf(" · %d sano(s) · %d carta(s) en mano", jug.Sanos(), len(jug.Mano()))
	if deTurno {
		encabezado += "  <- le toca"
	}
	pacientes := jug.Pacientes()
	if len(pacientes) == 0 {
		return c.pintar(ansiNegrita, encabezado) + "\n  (sin pacientes en el tablero)"
	}
	cartas := make([]string, 0, len(pacientes))
	for _, p := range pacientes {
		cartas = append(cartas, c.pacienteVirus(p))
	}
	return c.pintar(ansiNegrita, encabezado) + "\n" + unirLado(cartas, anchoMini)
}

// ListaPacientesVirus enumera los pacientes de un jugador para elegir uno.
func ListaPacientesVirus(jug *virus.Jugador) string {
	var b strings.Builder
	for i, p := range jug.Pacientes() {
		fmt.Fprintf(&b, "  [%d] %s (%s) - %s\n", i+1, p.Carta().Nombre(), p.Color(), estadoVirus(p))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// pacienteVirus dibuja un paciente del tablero con lo que tiene encima.
func (c *Consola) pacienteVirus(p *virus.PacienteEnJuego) string {
	lineas := []string{strings.ToUpper(p.Color().Equipo())}
	lineas = append(lineas, envolver(p.Carta().Nombre(), anchoMini)...)
	lineas = append(lineas, estadoVirus(p))
	encima := p.Encima()
	for _, e := range encima {
		lineas = append(lineas, envolver("+ "+e.Nombre(), anchoMini)...)
	}
	if p.Dormido() && len(encima) == 0 {
		lineas = append(lineas, "+ culpa del Coste")
	}
	return c.cartaDeAncho(lineas, ansiColorJuego[p.Color()], anchoMini)
}

// estadoVirus describe el estado de un paciente del juego.
func estadoVirus(p *virus.PacienteEnJuego) string {
	switch {
	case p.Estable():
		return "(^_^) ESTABLE"
	case p.Dormido():
		return "(-_-) zZz dormido"
	case p.Camas() > 0:
		return "(o_o) sano, con cama"
	default:
		return "(o_o) sano"
	}
}
