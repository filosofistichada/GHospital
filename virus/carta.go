package virus

// Color identifica el equipo de la costa de una carta.
type Color int

// Colores del juego. Multicolor es el comodín: combina con todos.
const (
	Rojo       Color = iota // Junior de Barranquilla
	Azul                    // Unión Magdalena
	Verde                   // Real Cartagena
	Amarillo                // Jaguares de Córdoba
	Multicolor              // Selección Colombia en el Metropolitano
)

// coloresEnOrden fija el orden en que se recorren y muestran los tableros
// (un map no tiene orden).
var coloresEnOrden = []Color{Rojo, Azul, Verde, Amarillo, Multicolor}

// String devuelve el color en texto legible.
func (c Color) String() string {
	switch c {
	case Rojo:
		return "rojo"
	case Azul:
		return "azul"
	case Verde:
		return "verde"
	case Amarillo:
		return "amarillo"
	case Multicolor:
		return "multicolor"
	default:
		return "desconocido"
	}
}

// Equipo devuelve el equipo que representa el color.
func (c Color) Equipo() string {
	switch c {
	case Rojo:
		return "Junior"
	case Azul:
		return "Unión Magdalena"
	case Verde:
		return "Real Cartagena"
	case Amarillo:
		return "Jaguares de Córdoba"
	case Multicolor:
		return "Selección Colombia"
	default:
		return "desconocido"
	}
}

// combinaCon indica si una carta de este color se puede jugar sobre un
// paciente del otro color: iguales, o alguno es multicolor.
func (c Color) combinaCon(otro Color) bool {
	return c == otro || c == Multicolor || otro == Multicolor
}

// TipoCarta agrupa las cartas por su papel en el juego.
type TipoCarta int

// Tipos de carta.
const (
	TipoPaciente TipoCarta = iota
	TipoAtaque
	TipoCama
	TipoTratamiento
)

// String devuelve el tipo en texto legible.
func (t TipoCarta) String() string {
	switch t {
	case TipoPaciente:
		return "paciente"
	case TipoAtaque:
		return "ataque de sueño"
	case TipoCama:
		return "cama"
	case TipoTratamiento:
		return "tratamiento"
	default:
		return "desconocido"
	}
}

// Requisitos dice qué objetivo necesita una carta para jugarse. La interfaz
// (consola o web) y los bots lo usan para saber qué preguntar o probar.
type Requisitos struct {
	Rival          bool // hay que elegir un rival
	PacienteRival  bool // hay que elegir un paciente del rival
	PacientePropio bool // hay que elegir un paciente propio
}

// Objetivo es a quién va dirigida una jugada. Cada carta usa solo los campos
// que pide en Requisitos.
type Objetivo struct {
	Rival          int   // índice del jugador rival
	PacienteRival  Color // color del paciente del rival
	PacientePropio Color // color de tu paciente
}

// Carta es cualquier carta del mazo. El juego llama a Jugar sin saber de qué
// tipo concreto es: cada carta sabe validar y aplicar su propio efecto, y
// también decide adónde va después (a tu tablero, encima de un paciente o
// al descarte).
//
// Jugar debe validar todo antes de modificar el juego: si devuelve error,
// el juego queda exactamente como estaba.
type Carta interface {
	Nombre() string
	Descripcion() string
	Color() Color
	Tipo() TipoCarta
	Requiere() Requisitos
	Jugar(j *Juego, obj Objetivo) error
}

// cartaBase tiene los datos comunes a todas las cartas. Cada tipo concreto
// la embebe y hereda Nombre, Descripcion y Color.
type cartaBase struct {
	nombre      string
	descripcion string
	color       Color
}

// Nombre devuelve el nombre de la carta.
func (b cartaBase) Nombre() string { return b.nombre }

// Descripcion explica qué hace la carta.
func (b cartaBase) Descripcion() string { return b.descripcion }

// Color devuelve el color (equipo) de la carta.
func (b cartaBase) Color() Color { return b.color }
