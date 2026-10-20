package virus

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

const (
	// cartasEnMano es cuántas cartas tiene cada jugador al final de su turno.
	cartasEnMano = 3
	// pacientesParaGanar es cuántos pacientes sanos de colores distintos se
	// necesitan para ganar.
	pacientesParaGanar = 4
)

// Juego es una partida de "Virus del Sueño". No es seguro para uso
// concurrente: quien lo comparta entre goroutines (la API web) debe
// protegerlo con su propio mutex.
type Juego struct {
	jugadores []*Jugador
	mazo      []Carta
	descarte  []Carta
	turno     int        // índice del jugador al que le toca
	ganador   *Jugador   // nil mientras nadie gane
	azar      *rand.Rand // inyectable: con la misma semilla, la misma partida
	mensajes  []string   // chistes y frases pendientes de mostrar
}

// NuevoJuego reparte una partida entre una persona y de 1 a 3 bots. El azar
// se recibe como parámetro para que los tests puedan fijar la semilla.
func NuevoJuego(nombre string, numBots int, azar *rand.Rand) (*Juego, error) {
	if numBots < 1 || numBots > len(perfilesBot) {
		return nil, ErrNumeroBots
	}
	j := &Juego{azar: azar, mazo: nuevoMazo()}
	j.jugadores = append(j.jugadores, nuevoJugador(nombre, nil))
	for i := 0; i < numBots; i++ {
		perfil := perfilesBot[i]
		j.jugadores = append(j.jugadores, nuevoJugador(perfil.nombre, &perfil))
	}
	j.azar.Shuffle(len(j.mazo), func(a, b int) { j.mazo[a], j.mazo[b] = j.mazo[b], j.mazo[a] })
	for _, jug := range j.jugadores {
		j.reponer(jug)
	}
	return j, nil
}

// Jugadores devuelve los jugadores en orden de turno (el 0 es la persona).
func (j *Juego) Jugadores() []*Jugador { return slices.Clone(j.jugadores) }

// Turno devuelve el índice del jugador al que le toca.
func (j *Juego) Turno() int { return j.turno }

// Ganador devuelve el ganador, o nil si la partida sigue.
func (j *Juego) Ganador() *Jugador { return j.ganador }

// CartasEnMazo devuelve cuántas cartas quedan por robar.
func (j *Juego) CartasEnMazo() int { return len(j.mazo) }

// TomarMensajes devuelve los chistes y frases acumulados y los borra.
func (j *Juego) TomarMensajes() []string {
	m := j.mensajes
	j.mensajes = nil
	return m
}

// JugarCarta juega la carta número indice de la mano del jugador de turno.
// El juego no sabe qué carta es: delega en su método Jugar. Si la jugada es
// válida, el jugador repone su mano y pasa el turno.
func (j *Juego) JugarCarta(indice int, obj Objetivo) error {
	if j.ganador != nil {
		return ErrJuegoTerminado
	}
	yo := j.actual()
	if indice < 0 || indice >= len(yo.mano) {
		return ErrCartaInvalida
	}
	carta := yo.mano[indice]
	if err := carta.Jugar(j, obj); err != nil {
		return fmt.Errorf("%s: %w", carta.Nombre(), err)
	}
	yo.mano = slices.Delete(yo.mano, indice, indice+1)
	j.terminarTurno()
	return nil
}

// Descartar bota de 1 a 3 cartas de la mano del jugador de turno, repone y
// pasa el turno.
func (j *Juego) Descartar(indices ...int) error {
	if j.ganador != nil {
		return ErrJuegoTerminado
	}
	yo := j.actual()
	if len(indices) < 1 || len(indices) > cartasEnMano {
		return ErrDescarteInvalido
	}
	vistos := make(map[int]bool)
	for _, i := range indices {
		if i < 0 || i >= len(yo.mano) || vistos[i] {
			return ErrDescarteInvalido
		}
		vistos[i] = true
	}
	var quedan []Carta
	for i, c := range yo.mano {
		if vistos[i] {
			j.descartar(c)
		} else {
			quedan = append(quedan, c)
		}
	}
	yo.mano = quedan
	j.terminarTurno()
	return nil
}

// Pasar termina el turno sin jugar. Solo se puede con la mano vacía, lo que
// pasa si se acabaron las cartas del mazo y del descarte.
func (j *Juego) Pasar() error {
	if j.ganador != nil {
		return ErrJuegoTerminado
	}
	if len(j.actual().mano) > 0 {
		return ErrTienesCartas
	}
	j.terminarTurno()
	return nil
}

// terminarTurno repone la mano, revisa si alguien ganó y, si no, pasa al
// siguiente jugador.
func (j *Juego) terminarTurno() {
	j.reponer(j.actual())
	// Se revisa a todos, empezando por quien jugó: un "Te montaron los
	// cachos" puede hacer ganar a otro.
	for i := range j.jugadores {
		jug := j.jugadores[(j.turno+i)%len(j.jugadores)]
		if jug.Sanos() >= pacientesParaGanar {
			j.ganador = jug
			j.anunciar("¡%s ganó con %d pacientes sanos! Que suene el picó.", jug.nombre, jug.Sanos())
			return
		}
	}
	j.turno = (j.turno + 1) % len(j.jugadores)
}

// reponer hace robar al jugador hasta tener cartasEnMano. Si el mazo se
// acaba, se baraja el descarte; si tampoco hay, se queda con menos.
func (j *Juego) reponer(jug *Jugador) {
	for len(jug.mano) < cartasEnMano {
		if len(j.mazo) == 0 {
			if len(j.descarte) == 0 {
				return
			}
			j.mazo, j.descarte = j.descarte, nil
			j.azar.Shuffle(len(j.mazo), func(a, b int) { j.mazo[a], j.mazo[b] = j.mazo[b], j.mazo[a] })
		}
		ultima := len(j.mazo) - 1
		jug.mano = append(jug.mano, j.mazo[ultima])
		j.mazo = j.mazo[:ultima]
	}
}

// ---------- Ayudas para las cartas ----------

// actual devuelve el jugador de turno.
func (j *Juego) actual() *Jugador { return j.jugadores[j.turno] }

// rivales devuelve a los demás jugadores, en orden a partir del siguiente.
func (j *Juego) rivales() []*Jugador {
	var rs []*Jugador
	for i := 1; i < len(j.jugadores); i++ {
		rs = append(rs, j.jugadores[(j.turno+i)%len(j.jugadores)])
	}
	return rs
}

// rival valida y devuelve el jugador rival de índice i.
func (j *Juego) rival(i int) (*Jugador, error) {
	if i < 0 || i >= len(j.jugadores) || i == j.turno {
		return nil, ErrRivalInvalido
	}
	return j.jugadores[i], nil
}

// pacienteDeRival valida y devuelve el rival y su paciente del objetivo.
func (j *Juego) pacienteDeRival(obj Objetivo) (*Jugador, *PacienteEnJuego, error) {
	rival, err := j.rival(obj.Rival)
	if err != nil {
		return nil, nil, err
	}
	p, ok := rival.tablero[obj.PacienteRival]
	if !ok {
		return nil, nil, ErrSinPaciente
	}
	return rival, p, nil
}

// atacar aplica un ataque (carta nil = ataque del Coste) a un paciente que ya
// se validó que no es estable: si tiene cama se anulan; si estaba libre se
// duerme; si ya dormía, sale del tablero.
func (j *Juego) atacar(victima *Jugador, p *PacienteEnJuego, carta Carta) {
	switch {
	case len(p.camas) > 0:
		ultima := len(p.camas) - 1
		j.descartar(p.camas[ultima], carta)
		p.camas = p.camas[:ultima]
	case !p.Dormido():
		p.ataques = append(p.ataques, carta)
	default:
		delete(victima.tablero, p.Color())
		j.descartar(p.todasLasCartas()...)
		j.descartar(carta)
		j.anunciar("¡Se fue pa' Barranquilla y perdió su silla! %s pierde a su hincha del %s.", victima.nombre, p.Color().Equipo())
	}
}

// descartar manda cartas a la pila de descarte (los nil se ignoran).
func (j *Juego) descartar(cartas ...Carta) {
	for _, c := range cartas {
		if c != nil {
			j.descarte = append(j.descarte, c)
		}
	}
}

// anunciar guarda un mensaje para que la interfaz lo muestre.
func (j *Juego) anunciar(formato string, args ...any) {
	j.mensajes = append(j.mensajes, fmt.Sprintf(formato, args...))
}
