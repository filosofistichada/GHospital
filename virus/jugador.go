package virus

import "slices"

// PacienteEnJuego es una carta de paciente puesta en un tablero, con las
// cartas que tiene encima. Nunca tiene ataques y camas a la vez, porque se
// anulan entre sí.
type PacienteEnJuego struct {
	carta   Carta
	ataques []Carta // un nil es un ataque del Coste, que no deja carta: deja la culpa
	camas   []Carta
}

// Carta devuelve la carta del paciente.
func (p *PacienteEnJuego) Carta() Carta { return p.carta }

// Color devuelve el color del paciente.
func (p *PacienteEnJuego) Color() Color { return p.carta.Color() }

// Dormido indica si tiene un ataque de sueño encima.
func (p *PacienteEnJuego) Dormido() bool { return len(p.ataques) > 0 }

// Camas devuelve cuántas camas tiene encima.
func (p *PacienteEnJuego) Camas() int { return len(p.camas) }

// Estable indica si tiene dos camas: es inmune a todo.
func (p *PacienteEnJuego) Estable() bool { return len(p.camas) >= 2 }

// Sano indica si cuenta para ganar: despierto (con o sin camas).
func (p *PacienteEnJuego) Sano() bool { return !p.Dormido() }

// Libre indica si no tiene nada encima.
func (p *PacienteEnJuego) Libre() bool { return len(p.ataques) == 0 && len(p.camas) == 0 }

// Encima devuelve las cartas que tiene encima (los ataques del Coste no
// aparecen porque no son cartas).
func (p *PacienteEnJuego) Encima() []Carta {
	var cartas []Carta
	for _, c := range append(slices.Clone(p.ataques), p.camas...) {
		if c != nil {
			cartas = append(cartas, c)
		}
	}
	return cartas
}

// todasLasCartas devuelve el paciente y lo que tiene encima, para descartar.
func (p *PacienteEnJuego) todasLasCartas() []Carta {
	return append([]Carta{p.carta}, p.Encima()...)
}

// perfilBot es la personalidad de un bot: su nombre y sus frases.
type perfilBot struct {
	nombre string
	frases []string
}

// perfilesBot son los bots disponibles, en el orden en que entran a jugar.
var perfilesBot = []perfilBot{
	{"La Seño Eneida", []string{
		"Yo no digo nada, pero ahí le dejo eso.",
		"Mija, a su paciente lo vieron dormido en otra hamaca.",
		"Eso le pasa por no invitarme al sancocho.",
	}},
	{"El Compa Jhonier", []string{
		"¡Súbele al picó que esto se prendió!",
		"Tranquilo, viejo, que yo sabía que iba a perder el Junior.",
		"Eso no es trampa, eso es malicia indígena.",
	}},
	{"Tía Nury", []string{
		"En mis tiempos uno se dormía temprano y sin narcolepsia.",
		"Tomé aguapanela y aquí estoy, mijo.",
		"Ese paciente necesita es una mañana de rosario.",
	}},
}

// Jugador es una persona o un bot sentado a la mesa.
type Jugador struct {
	nombre  string
	bot     *perfilBot // nil si es una persona
	mano    []Carta
	tablero map[Color]*PacienteEnJuego // la clave impide dos pacientes del mismo color
}

// nuevoJugador crea un jugador sin cartas.
func nuevoJugador(nombre string, bot *perfilBot) *Jugador {
	return &Jugador{nombre: nombre, bot: bot, tablero: make(map[Color]*PacienteEnJuego)}
}

// Nombre devuelve el nombre del jugador.
func (j *Jugador) Nombre() string { return j.nombre }

// EsBot indica si el jugador lo maneja el computador.
func (j *Jugador) EsBot() bool { return j.bot != nil }

// Mano devuelve una copia de las cartas en la mano.
func (j *Jugador) Mano() []Carta { return slices.Clone(j.mano) }

// Pacientes devuelve los pacientes del tablero en orden de color.
func (j *Jugador) Pacientes() []*PacienteEnJuego {
	var pacientes []*PacienteEnJuego
	for _, c := range coloresEnOrden {
		if p, ok := j.tablero[c]; ok {
			pacientes = append(pacientes, p)
		}
	}
	return pacientes
}

// Sanos cuenta los pacientes que no están dormidos. Como no puede haber dos
// del mismo color y el comodín vale por el que falte, con 4 sanos se gana.
func (j *Jugador) Sanos() int {
	n := 0
	for _, p := range j.tablero {
		if p.Sano() {
			n++
		}
	}
	return n
}
