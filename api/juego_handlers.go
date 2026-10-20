package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"costenos-narcolepsia/virus"
)

// errSinPartida se responde con 404 si todavía no se ha empezado un juego.
var errSinPartida = errors.New("no hay partida: empieza una nueva, que la mesa ya está puesta")

// errNoEsTuTurno se responde si la persona intenta jugar en turno de un bot
// (no debería pasar, porque el servidor juega los bots de una vez).
var errNoEsTuTurno = errors.New("no es tu turno: los bots todavía están pensando")

// POST /api/juego/nuevo: empieza una partida.
func (s *Servidor) nuevoJuego(w http.ResponseWriter, r *http.Request) {
	var req nuevoJuegoReq
	if err := leerJSON(r, &req); err != nil {
		escribirError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Nombre) == "" {
		req.Nombre = "Sofía"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := virus.NuevoJuego(req.Nombre, req.Bots, s.cfg.NuevoAzar())
	if err != nil {
		escribirError(w, http.StatusBadRequest, err)
		return
	}
	s.juego = j
	escribirJSON(w, http.StatusOK, s.juegoLocked([]string{"¡Que empiece el juego! Cuidado con el Coste, que anda suelto."}))
}

// GET /api/juego: estado de la partida actual.
func (s *Servidor) verJuego(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.juego == nil {
		escribirError(w, http.StatusNotFound, errSinPartida)
		return
	}
	escribirJSON(w, http.StatusOK, s.juegoLocked(nil))
}

// POST /api/juego/jugar: la persona juega una carta y luego juegan los bots.
func (s *Servidor) jugarCarta(w http.ResponseWriter, r *http.Request) {
	var req jugadaReq
	if err := leerJSON(r, &req); err != nil {
		escribirError(w, http.StatusBadRequest, err)
		return
	}
	s.accionDePersona(w, func(j *virus.Juego) (string, error) {
		mano := j.Jugadores()[0].Mano()
		if req.Carta < 0 || req.Carta >= len(mano) {
			return "", virus.ErrCartaInvalida
		}
		nombre := mano[req.Carta].Nombre()
		err := j.JugarCarta(req.Carta, virus.Objetivo{
			Rival:          req.Rival,
			PacienteRival:  colorDesdeTexto(req.PacienteRival),
			PacientePropio: colorDesdeTexto(req.PacientePropio),
		})
		return "Jugaste " + nombre + ".", err
	})
}

// POST /api/juego/descartar: la persona descarta y luego juegan los bots.
// Con la lista vacía y la mano vacía, pasa el turno.
func (s *Servidor) descartar(w http.ResponseWriter, r *http.Request) {
	var req descarteReq
	if err := leerJSON(r, &req); err != nil {
		escribirError(w, http.StatusBadRequest, err)
		return
	}
	s.accionDePersona(w, func(j *virus.Juego) (string, error) {
		if len(req.Cartas) == 0 {
			return "Pasaste el turno.", j.Pasar()
		}
		return fmt.Sprintf("Descartaste %d carta(s).", len(req.Cartas)), j.Descartar(req.Cartas...)
	})
}

// accionDePersona valida que sea el turno de la persona, aplica la acción y,
// si fue válida, juega los turnos de los bots hasta que le vuelva a tocar o
// alguien gane. Responde con todos los mensajes en orden.
func (s *Servidor) accionDePersona(w http.ResponseWriter, accion func(*virus.Juego) (string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.juego
	if j == nil {
		escribirError(w, http.StatusNotFound, errSinPartida)
		return
	}
	if j.Ganador() != nil {
		escribirError(w, http.StatusConflict, virus.ErrJuegoTerminado)
		return
	}
	if j.Turno() != 0 {
		escribirError(w, http.StatusConflict, errNoEsTuTurno)
		return
	}
	mensaje, err := accion(j)
	if err != nil {
		escribirError(w, http.StatusConflict, err)
		return
	}
	mensajes := append([]string{mensaje}, j.TomarMensajes()...)
	for j.Ganador() == nil && j.Jugadores()[j.Turno()].EsBot() {
		if err := j.JugarTurnoBot(); err != nil {
			escribirError(w, http.StatusInternalServerError, err)
			return
		}
		mensajes = append(mensajes, j.TomarMensajes()...)
	}
	escribirJSON(w, http.StatusOK, s.juegoLocked(mensajes))
}
