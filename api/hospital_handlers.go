package api

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"

	"costenos-narcolepsia/hospital"
)

// errPacienteNoEncontrado se responde con 404 cuando el ID no existe.
var errPacienteNoEncontrado = errors.New("ese paciente no existe en este hospital: ni en la lista del Coste")

// errDoctorNoEncontrado se responde con 404 cuando el ID no es de un doctor.
var errDoctorNoEncontrado = errors.New("ese doctor no trabaja aquí: el Dr. Escalona sigue buscando la 102")

// ataqueReq es el cuerpo opcional de POST /api/pacientes/{id}/ataque.
type ataqueReq struct {
	Ubicacion string `json:"ubicacion"`
}

// GET /api/estado: todo lo que la página necesita para dibujarse.
func (s *Servidor) verEstado(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	escribirJSON(w, http.StatusOK, s.estadoLocked())
}

// GET /api/pasillo: consulta PacientesEnPasillo.
func (s *Servidor) verPasillo(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pasillo := []pacienteDTO{}
	for _, p := range s.h.PacientesEnPasillo() {
		pasillo = append(pasillo, s.pacienteDTO(p, 0))
	}
	escribirJSON(w, http.StatusOK, pasillo)
}

// GET /api/doctores/{id}/episodios: consulta MisEpisodios.
func (s *Servidor) verEpisodiosDoctor(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.h.Doctores() {
		if d.ID() != r.PathValue("id") {
			continue
		}
		episodios := []episodioDTO{}
		for _, e := range d.MisEpisodios() {
			episodios = append(episodios, s.episodioDTO(e))
		}
		escribirJSON(w, http.StatusOK, episodios)
		return
	}
	escribirError(w, http.StatusNotFound, errDoctorNoEncontrado)
}

// GET /api/severos: consulta ReporteSeveros.
func (s *Servidor) verSeveros(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	escribirJSON(w, http.StatusOK, s.severosLocked(s.habitacionesPorPacienteLocked()))
}

// POST /api/pacientes/{id}/ataque: RegistrarEpisodio. Quedarse sin cama no
// es un fallo de la petición (el episodio se registra igual), así que se
// responde 200 con un aviso.
func (s *Servidor) registrarAtaque(w http.ResponseWriter, r *http.Request) {
	var req ataqueReq
	if err := leerJSON(r, &req); err != nil {
		escribirError(w, http.StatusBadRequest, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pacienteLocked(r.PathValue("id"))
	if p == nil {
		escribirError(w, http.StatusNotFound, errPacienteNoEncontrado)
		return
	}
	if req.Ubicacion == "" {
		req.Ubicacion = s.cfg.Lugares[rand.IntN(len(s.cfg.Lugares))]
	}
	err := s.h.RegistrarEpisodio(p, req.Ubicacion)
	switch {
	case err == nil:
		s.responderAccion(w, fmt.Sprintf("%s se durmió %s y consiguió cama en la %s.", p.Nombre(), req.Ubicacion, p.Ubicacion()), "")
	case errors.Is(err, hospital.ErrSinHabitacion):
		s.responderAccion(w, fmt.Sprintf("%s se durmió %s y se quedó en el pasillo.", p.Nombre(), req.Ubicacion), err.Error())
	default:
		escribirError(w, http.StatusConflict, err)
	}
}

// POST /api/pacientes/{id}/despertar: DespertarPaciente.
func (s *Servidor) despertar(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pacienteLocked(r.PathValue("id"))
	if p == nil {
		escribirError(w, http.StatusNotFound, errPacienteNoEncontrado)
		return
	}
	if err := s.h.DespertarPaciente(p); err != nil {
		escribirError(w, http.StatusConflict, err)
		return
	}
	s.responderAccion(w, fmt.Sprintf("%s se despertó.", p.Nombre()), "")
}

// POST /api/pacientes/{id}/cama: AsignarHabitacion.
func (s *Servidor) asignarCama(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pacienteLocked(r.PathValue("id"))
	if p == nil {
		escribirError(w, http.StatusNotFound, errPacienteNoEncontrado)
		return
	}
	hab, err := s.h.AsignarHabitacion(p)
	if err != nil {
		escribirError(w, http.StatusConflict, err)
		return
	}
	s.responderAccion(w, fmt.Sprintf("%s ya está roncando en la habitación %d.", p.Nombre(), hab.Numero()), "")
}

// POST /api/escenario/reiniciar: vuelve a armar el hospital del escenario.
func (s *Servidor) reiniciar(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.h, s.equipos = s.cfg.ArmarHospital()
	s.responderAccion(w, "Hospital reiniciado: todos despiertos y las camas tendidas.", "")
}

// responderAccion responde 200 con el mensaje y el estado nuevo.
func (s *Servidor) responderAccion(w http.ResponseWriter, mensaje, aviso string) {
	escribirJSON(w, http.StatusOK, accionDTO{Mensaje: mensaje, Aviso: aviso, Estado: s.estadoLocked()})
}

// pacienteLocked busca un paciente por ID, o devuelve nil.
func (s *Servidor) pacienteLocked(id string) *hospital.Paciente {
	for _, p := range s.h.Pacientes() {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// habitacionesPorPacienteLocked dice en qué habitación está cada paciente.
func (s *Servidor) habitacionesPorPacienteLocked() map[*hospital.Paciente]int {
	habDe := make(map[*hospital.Paciente]int)
	for _, hab := range s.h.Habitaciones() {
		for _, p := range hab.Ocupantes() {
			habDe[p] = hab.Numero()
		}
	}
	return habDe
}
