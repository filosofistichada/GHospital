package virus

import (
	"cmp"
	"fmt"
	"slices"
)

// prioridadBot ordena los tipos de carta según lo que el bot prefiere hacer:
// curar lo propio, atacar, poner pacientes y, por último, tratamientos.
var prioridadBot = map[TipoCarta]int{
	TipoCama:        0,
	TipoAtaque:      1,
	TipoPaciente:    2,
	TipoTratamiento: 3,
}

// JugarTurnoBot juega el turno del bot de turno: prueba sus cartas en orden
// de prioridad con cada objetivo posible y juega la primera jugada válida.
// Si ninguna sirve, descarta toda la mano. Como Jugar no cambia nada cuando
// falla, probar es seguro.
func (j *Juego) JugarTurnoBot() error {
	if j.ganador != nil {
		return ErrJuegoTerminado
	}
	yo := j.actual()
	if yo.bot == nil {
		return ErrNoEsBot
	}
	if len(yo.mano) == 0 {
		return j.Pasar()
	}

	indices := make([]int, len(yo.mano))
	for i := range indices {
		indices[i] = i
	}
	slices.SortStableFunc(indices, func(a, b int) int {
		return cmp.Compare(prioridadBot[yo.mano[a].Tipo()], prioridadBot[yo.mano[b].Tipo()])
	})

	for _, i := range indices {
		carta := yo.mano[i]
		for _, obj := range j.objetivosPosibles(carta.Requiere()) {
			antes := len(j.mensajes)
			if j.JugarCarta(i, obj) == nil {
				// La frase va antes de los mensajes que produjo la jugada.
				frase := fmt.Sprintf("%s jugó %s. \"%s\"", yo.nombre, carta.Nombre(), yo.bot.frases[j.azar.IntN(len(yo.bot.frases))])
				j.mensajes = slices.Insert(j.mensajes, antes, frase)
				return nil
			}
		}
	}
	todas := make([]int, len(yo.mano))
	for i := range todas {
		todas[i] = i
	}
	j.anunciar("%s no tenía nada que jugar y botó su mano.", yo.nombre)
	return j.Descartar(todas...)
}

// objetivosPosibles arma los objetivos a probar para unos requisitos, en el
// orden en que el bot los prefiere: rivales que van ganando primero, y sus
// propios pacientes dormidos primero (luego los que tienen menos camas).
func (j *Juego) objetivosPosibles(req Requisitos) []Objetivo {
	rivales := []int{-1}
	if req.Rival {
		rivales = rivales[:0]
		for i := range j.jugadores {
			if i != j.turno {
				rivales = append(rivales, i)
			}
		}
		slices.SortStableFunc(rivales, func(a, b int) int {
			return cmp.Compare(j.jugadores[b].Sanos(), j.jugadores[a].Sanos())
		})
	}

	propios := []Color{Multicolor}
	if req.PacientePropio {
		propios = propios[:0]
		mios := j.actual().Pacientes()
		slices.SortStableFunc(mios, func(a, b *PacienteEnJuego) int {
			if a.Dormido() != b.Dormido() {
				if a.Dormido() {
					return -1
				}
				return 1
			}
			return cmp.Compare(a.Camas(), b.Camas())
		})
		for _, p := range mios {
			propios = append(propios, p.Color())
		}
	}

	var objetivos []Objetivo
	for _, r := range rivales {
		colorRival := []Color{Multicolor}
		if req.PacienteRival {
			colorRival = colorRival[:0]
			for _, p := range j.jugadores[r].Pacientes() {
				colorRival = append(colorRival, p.Color())
			}
		}
		for _, cr := range colorRival {
			for _, cp := range propios {
				objetivos = append(objetivos, Objetivo{Rival: r, PacienteRival: cr, PacientePropio: cp})
			}
		}
	}
	return objetivos
}
