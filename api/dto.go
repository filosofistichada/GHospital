package api

import (
	"slices"
	"strings"

	"costenos-narcolepsia/hospital"
	"costenos-narcolepsia/virus"
)

// Los DTO (objetos de transferencia) son la forma JSON de los datos. Viven
// aquí y no en hospital ni en virus: el modelo no sabe que existe una web.

type errorDTO struct {
	Error string `json:"error"`
}

// ---------- Hospital ----------

type pacienteDTO struct {
	ID         string `json:"id"`
	Nombre     string `json:"nombre"`
	Edad       int    `json:"edad"`
	Nivel      string `json:"nivel"`
	Estado     string `json:"estado"`       // texto legible
	Codigo     string `json:"codigo"`       // despierto | pasillo | cama
	Ubicacion  string `json:"ubicacion"`    // último lugar conocido
	Doctor     string `json:"doctor"`       // vacío si no lo ha diagnosticado nadie
	Equipo     string `json:"equipo"`       // color del equipo: rojo, azul, verde, amarillo, multicolor
	Habitacion int    `json:"habitacion"`   // 0 si no está en una habitación
	Episodios  int    `json:"episodiosHoy"` // solo se llena en el reporte de severos
}

type episodioDTO struct {
	ID         string `json:"id"`
	Hora       string `json:"hora"`
	PacienteID string `json:"pacienteId"`
	Paciente   string `json:"paciente"`
	Equipo     string `json:"equipo"`
	Ubicacion  string `json:"ubicacion"`
	Atendedor  string `json:"atendedor"`
	Habitacion int    `json:"habitacion"` // 0 si se quedó en el pasillo
	Resumen    string `json:"resumen"`
}

type habitacionDTO struct {
	Numero     int      `json:"numero"`
	Capacidad  int      `json:"capacidad"`
	Estado     string   `json:"estado"`
	Ocupantes  []string `json:"ocupantes"` // IDs de pacientes
	Disponible bool     `json:"disponible"`
}

type personalDTO struct {
	ID           string `json:"id"`
	Nombre       string `json:"nombre"`
	Rol          string `json:"rol"` // doctor | apoyo
	Especialidad string `json:"especialidad,omitempty"`
	Disponible   bool   `json:"disponible"`
}

type doctorDTO struct {
	ID           string        `json:"id"`
	Nombre       string        `json:"nombre"`
	Especialidad string        `json:"especialidad"`
	Episodios    []episodioDTO `json:"episodios"`
}

type estadoDTO struct {
	Nombre       string          `json:"nombre"`
	Pacientes    []pacienteDTO   `json:"pacientes"`
	Habitaciones []habitacionDTO `json:"habitaciones"`
	Personal     []personalDTO   `json:"personal"`
	Doctores     []doctorDTO     `json:"doctores"`
	Pasillo      []pacienteDTO   `json:"pasillo"`
	Severos      []pacienteDTO   `json:"severos"`
	Historial    []episodioDTO   `json:"historial"`
}

type accionDTO struct {
	Mensaje string    `json:"mensaje"`
	Aviso   string    `json:"aviso,omitempty"` // por ejemplo, no hubo cama
	Estado  estadoDTO `json:"estado"`
}

// estadoLocked arma el estado completo. Se llama con s.mu tomado.
func (s *Servidor) estadoLocked() estadoDTO {
	h := s.h
	habDe := make(map[*hospital.Paciente]int)
	var habitaciones []habitacionDTO
	for _, hab := range h.Habitaciones() {
		dto := habitacionDTO{
			Numero:     hab.Numero(),
			Capacidad:  hab.Capacidad(),
			Estado:     hab.Estado().String(),
			Ocupantes:  []string{},
			Disponible: hab.EstaDisponible(),
		}
		for _, p := range hab.Ocupantes() {
			habDe[p] = hab.Numero()
			dto.Ocupantes = append(dto.Ocupantes, p.ID())
		}
		habitaciones = append(habitaciones, dto)
	}

	var pacientes []pacienteDTO
	for _, p := range h.Pacientes() {
		pacientes = append(pacientes, s.pacienteDTO(p, habDe[p]))
	}

	especialidades := make(map[string]string) // ID del doctor -> especialidad
	var doctores []doctorDTO
	for _, d := range h.Doctores() {
		especialidades[d.ID()] = d.Especialidad()
		dto := doctorDTO{ID: d.ID(), Nombre: d.Nombre(), Especialidad: d.Especialidad(), Episodios: []episodioDTO{}}
		for _, r := range d.MisEpisodios() {
			dto.Episodios = append(dto.Episodios, s.episodioDTO(r))
		}
		doctores = append(doctores, dto)
	}

	var personal []personalDTO
	for _, a := range h.Personal() {
		dto := personalDTO{ID: a.ID(), Nombre: a.Nombre(), Rol: "apoyo", Disponible: a.Disponible()}
		if esp, esDoctor := especialidades[a.ID()]; esDoctor {
			dto.Rol, dto.Especialidad = "doctor", esp
		}
		personal = append(personal, dto)
	}

	pasillo := []pacienteDTO{}
	for _, p := range h.PacientesEnPasillo() {
		pasillo = append(pasillo, s.pacienteDTO(p, 0))
	}

	historial := []episodioDTO{}
	for _, r := range h.Historial() {
		historial = append(historial, s.episodioDTO(r))
	}

	return estadoDTO{
		Nombre:       h.Nombre(),
		Pacientes:    pacientes,
		Habitaciones: habitaciones,
		Personal:     personal,
		Doctores:     doctores,
		Pasillo:      pasillo,
		Severos:      s.severosLocked(habDe),
		Historial:    historial,
	}
}

// severosLocked convierte el map de ReporteSeveros en una lista ordenada por
// ID (JSON no puede usar punteros como claves y un map no tiene orden).
func (s *Servidor) severosLocked(habDe map[*hospital.Paciente]int) []pacienteDTO {
	severos := []pacienteDTO{}
	for p, n := range s.h.ReporteSeveros() {
		dto := s.pacienteDTO(p, habDe[p])
		dto.Episodios = n
		severos = append(severos, dto)
	}
	slices.SortFunc(severos, func(a, b pacienteDTO) int { return strings.Compare(a.ID, b.ID) })
	return severos
}

// pacienteDTO convierte un paciente; hab es su habitación (0 si ninguna).
func (s *Servidor) pacienteDTO(p *hospital.Paciente, hab int) pacienteDTO {
	codigo := "despierto"
	switch p.Estado() {
	case hospital.DormidoEnPasillo:
		codigo = "pasillo"
	case hospital.DormidoEnCama:
		codigo = "cama"
	}
	doctor := ""
	if d := p.DoctorAsignado(); d != nil {
		doctor = d.Nombre()
	}
	return pacienteDTO{
		ID:         p.ID(),
		Nombre:     p.Nombre(),
		Edad:       p.Edad(),
		Nivel:      p.Nivel().String(),
		Estado:     p.Estado().String(),
		Codigo:     codigo,
		Ubicacion:  p.Ubicacion(),
		Doctor:     doctor,
		Equipo:     s.equipoDe(p.ID()),
		Habitacion: hab,
	}
}

// episodioDTO convierte un registro de episodio.
func (s *Servidor) episodioDTO(r hospital.RegistroEpisodio) episodioDTO {
	hab := 0
	if h := r.Habitacion(); h != nil {
		hab = h.Numero()
	}
	return episodioDTO{
		ID:         r.ID(),
		Hora:       r.FechaHora().Format("15:04:05"),
		PacienteID: r.Paciente().ID(),
		Paciente:   r.Paciente().Nombre(),
		Equipo:     s.equipoDe(r.Paciente().ID()),
		Ubicacion:  r.Ubicacion(),
		Atendedor:  r.Atendedor().Nombre(),
		Habitacion: hab,
		Resumen:    r.Resumen(),
	}
}

// equipoDe devuelve el color del equipo del paciente (comodín si no tiene).
func (s *Servidor) equipoDe(id string) string {
	if e, ok := s.equipos[id]; ok {
		return e
	}
	return "multicolor"
}

// ---------- Juego ----------

type requisitosDTO struct {
	Rival          bool `json:"rival"`
	PacienteRival  bool `json:"pacienteRival"`
	PacientePropio bool `json:"pacientePropio"`
}

type cartaDTO struct {
	Indice      int           `json:"indice"`
	Nombre      string        `json:"nombre"`
	Descripcion string        `json:"descripcion"`
	Tipo        string        `json:"tipo"`  // paciente | ataque | cama | tratamiento
	Icono       string        `json:"icono"` // nombre del dibujo que usa la web
	Color       string        `json:"color"`
	Requiere    requisitosDTO `json:"requiere"`
}

type pacienteJuegoDTO struct {
	Color      string   `json:"color"`
	Equipo     string   `json:"equipo"`
	Nombre     string   `json:"nombre"`
	Dormido    bool     `json:"dormido"`
	Camas      int      `json:"camas"`
	Estable    bool     `json:"estable"`
	Encima     []string `json:"encima"`
	CulpaCoste bool     `json:"culpaCoste"` // dormido por el Coste (sin carta encima)
}

type jugadorDTO struct {
	Indice    int                `json:"indice"`
	Nombre    string             `json:"nombre"`
	EsBot     bool               `json:"esBot"`
	Sanos     int                `json:"sanos"`
	Cartas    int                `json:"cartas"`
	Pacientes []pacienteJuegoDTO `json:"pacientes"`
}

type juegoDTO struct {
	Jugadores []jugadorDTO `json:"jugadores"`
	Turno     int          `json:"turno"`
	Ganador   string       `json:"ganador"` // vacío mientras nadie gane
	Mano      []cartaDTO   `json:"mano"`    // solo la de la persona (jugador 0)
	Mazo      int          `json:"mazo"`
	Mensajes  []string     `json:"mensajes"`
}

type nuevoJuegoReq struct {
	Nombre string `json:"nombre"`
	Bots   int    `json:"bots"`
}

type jugadaReq struct {
	Carta          int    `json:"carta"`
	Rival          int    `json:"rival"`
	PacienteRival  string `json:"pacienteRival"`
	PacientePropio string `json:"pacientePropio"`
}

type descarteReq struct {
	Cartas []int `json:"cartas"`
}

// coloresJuego permite traducir el texto del JSON a virus.Color.
var coloresJuego = []virus.Color{virus.Rojo, virus.Azul, virus.Verde, virus.Amarillo, virus.Multicolor}

// colorDesdeTexto traduce "rojo" a virus.Rojo. Un texto vacío o desconocido
// da Multicolor, y la carta decide si ese objetivo le sirve.
func colorDesdeTexto(texto string) virus.Color {
	for _, c := range coloresJuego {
		if c.String() == texto {
			return c
		}
	}
	return virus.Multicolor
}

// juegoLocked arma el estado de la partida. Se llama con s.mu tomado.
func (s *Servidor) juegoLocked(mensajes []string) juegoDTO {
	j := s.juego
	dto := juegoDTO{Turno: j.Turno(), Mazo: j.CartasEnMazo(), Mensajes: mensajes, Mano: []cartaDTO{}}
	if mensajes == nil {
		dto.Mensajes = []string{}
	}
	if g := j.Ganador(); g != nil {
		dto.Ganador = g.Nombre()
	}
	for i, jug := range j.Jugadores() {
		jd := jugadorDTO{Indice: i, Nombre: jug.Nombre(), EsBot: jug.EsBot(), Sanos: jug.Sanos(), Cartas: len(jug.Mano()), Pacientes: []pacienteJuegoDTO{}}
		for _, p := range jug.Pacientes() {
			encima := []string{}
			for _, c := range p.Encima() {
				encima = append(encima, c.Nombre())
			}
			jd.Pacientes = append(jd.Pacientes, pacienteJuegoDTO{
				Color:      p.Color().String(),
				Equipo:     p.Color().Equipo(),
				Nombre:     p.Carta().Nombre(),
				Dormido:    p.Dormido(),
				Camas:      p.Camas(),
				Estable:    p.Estable(),
				Encima:     encima,
				CulpaCoste: p.Dormido() && len(encima) == 0,
			})
		}
		dto.Jugadores = append(dto.Jugadores, jd)
	}
	for i, c := range j.Jugadores()[0].Mano() {
		req := c.Requiere()
		dto.Mano = append(dto.Mano, cartaDTO{
			Indice:      i,
			Nombre:      c.Nombre(),
			Descripcion: c.Descripcion(),
			Tipo:        tipoDTO(c.Tipo()),
			Icono:       iconoDeCarta(c),
			Color:       c.Color().String(),
			Requiere:    requisitosDTO{Rival: req.Rival, PacienteRival: req.PacienteRival, PacientePropio: req.PacientePropio},
		})
	}
	return dto
}

// tipoDTO da el código corto de cada tipo de carta para el CSS.
func tipoDTO(t virus.TipoCarta) string {
	switch t {
	case virus.TipoPaciente:
		return "paciente"
	case virus.TipoAtaque:
		return "ataque"
	case virus.TipoCama:
		return "cama"
	default:
		return "tratamiento"
	}
}

// iconoDeCarta elige el dibujo de la carta. Aquí sí se mira el tipo
// concreto, y está bien: es una decisión de presentación (qué imagen
// mostrar), no una regla del juego. Las reglas se despachan por la
// interfaz Carta, sin type switch.
func iconoDeCarta(c virus.Carta) string {
	switch c.(type) {
	case *virus.CartaCoste:
		return "coste"
	case *virus.CartaDoctor:
		return "doctor"
	case *virus.CartaCamillero:
		return "camillero"
	case *virus.CartaAtaque:
		return "comida"
	case *virus.CartaCama:
		return "cama"
	case *virus.CartaPaciente:
		return "equipo-" + c.Color().String()
	default:
		return "tratamiento"
	}
}
