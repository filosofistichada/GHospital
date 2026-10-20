package hospital

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// Hospital coordina pacientes, personal y habitaciones, y guarda el
// historial de todos los episodios de sueño.
//
// Hospital es seguro para uso concurrente: todos sus métodos exportados
// toman el mutex. Los métodos de Paciente, Doctor y Habitacion no lo son por
// sí solos, así que mientras haya goroutines activas todo cambio de estado
// debe pasar por el Hospital.
type Hospital struct {
	mu           sync.Mutex // protege todos los campos de abajo y el estado de pacientes, doctores y habitaciones
	nombre       string
	pacientes    []*Paciente
	doctores     []*Doctor
	personal     []Atendedor // doctores, camilleros y cualquier otro Atendedor
	habitaciones []*Habitacion
	historial    []*RegistroEpisodio
	turno        int              // índice del próximo atendedor en el turno rotativo
	ahora        func() time.Time // reloj del hospital; los tests lo reemplazan
}

// NuevoHospital crea un hospital vacío.
func NuevoHospital(nombre string) *Hospital {
	return &Hospital{nombre: nombre, ahora: time.Now}
}

// Nombre devuelve el nombre del hospital. No toma el lock porque el nombre
// no cambia después de crear el hospital.
func (h *Hospital) Nombre() string { return h.nombre }

// AdmitirPaciente registra un paciente. Falla si es nil o si ya hay un
// paciente con el mismo ID.
func (h *Hospital) AdmitirPaciente(p *Paciente) error {
	if p == nil {
		return ErrPacienteNulo
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, q := range h.pacientes {
		if q.ID() == p.ID() {
			return fmt.Errorf("admitir %s: %w", p.ID(), ErrPacienteDuplicado)
		}
	}
	h.pacientes = append(h.pacientes, p)
	return nil
}

// ContratarDoctor agrega un doctor a la lista de doctores y al personal que
// se despacha.
func (h *Hospital) ContratarDoctor(d *Doctor) error {
	// Se revisa aquí porque un *Doctor nil guardado en una interfaz ya no es
	// una interfaz nil, y contratarPersonalLocked no lo detectaría.
	if d == nil {
		return ErrPersonalNulo
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.contratarPersonalLocked(d); err != nil {
		return err
	}
	h.doctores = append(h.doctores, d)
	return nil
}

// ContratarPersonal agrega cualquier Atendedor al personal. Falla si es nil
// o si ya hay alguien con el mismo ID.
func (h *Hospital) ContratarPersonal(a Atendedor) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.contratarPersonalLocked(a)
}

// AgregarHabitacion registra una habitación. Falla si es nil o si el número
// ya existe.
func (h *Hospital) AgregarHabitacion(hab *Habitacion) error {
	if hab == nil {
		return ErrHabitacionNula
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, otra := range h.habitaciones {
		if otra.Numero() == hab.Numero() {
			return fmt.Errorf("habitación %d: %w", hab.Numero(), ErrHabitacionDuplicada)
		}
	}
	h.habitaciones = append(h.habitaciones, hab)
	return nil
}

// RegistrarEpisodio procesa un ataque de sueño completo:
//  1. el paciente se duerme en la ubicación (queda en el pasillo),
//  2. se despacha al siguiente atendedor disponible,
//  3. el atendedor lo atiende y devuelve el registro,
//  4. el registro se guarda en el historial (y en el del atendedor, si lleva),
//  5. se intenta asignar habitación.
//
// Si no hay cama, el episodio queda registrado sin habitación, el paciente
// sigue en el pasillo y el error devuelto envuelve ErrSinHabitacion.
func (h *Hospital) RegistrarEpisodio(p *Paciente, ubicacion string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.admitidoLocked(p) {
		return fmt.Errorf("registrar episodio: %w", ErrPacienteNoAdmitido)
	}
	if err := p.SufrirAtaqueDeSueno(ubicacion); err != nil {
		return fmt.Errorf("registrar episodio de %s: %w", p.Nombre(), err)
	}
	a, err := h.siguienteAtendedorLocked()
	if err != nil {
		return fmt.Errorf("registrar episodio de %s: %w", p.Nombre(), err)
	}
	registro, err := a.Atender(p, ubicacion)
	if err != nil {
		return fmt.Errorf("registrar episodio de %s: %w", p.Nombre(), err)
	}
	registro.id = fmt.Sprintf("E-%03d", len(h.historial)+1)
	r := &registro
	h.historial = append(h.historial, r)
	// No se pregunta por el tipo concreto, sino por la capacidad de llevar
	// historial: el atendedor recibe el mismo puntero que el hospital.
	if lh, ok := a.(llevaHistorial); ok {
		lh.anotarEpisodio(r)
	}
	// Se usa la versión Locked: llamar a AsignarHabitacion aquí intentaría
	// tomar otra vez el mutex que ya tenemos y el programa se bloquearía.
	_, err = h.asignarHabitacionLocked(p)
	return err
}

// AsignarHabitacion busca la primera habitación disponible y acuesta ahí al
// paciente dormido: la habitación cuenta al ocupante y el paciente pasa a
// DormidoEnCama. Si su último episodio no tenía habitación, se le anota.
// Si no hay cama, devuelve un error que envuelve ErrSinHabitacion y el
// paciente sigue en el pasillo.
func (h *Hospital) AsignarHabitacion(p *Paciente) (*Habitacion, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.asignarHabitacionLocked(p)
}

// DespertarPaciente despierta al paciente y, si estaba en una habitación, la
// libera para el siguiente.
func (h *Hospital) DespertarPaciente(p *Paciente) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.admitidoLocked(p) {
		return fmt.Errorf("despertar: %w", ErrPacienteNoAdmitido)
	}
	if err := p.Despertar(); err != nil {
		return fmt.Errorf("despertar a %s: %w", p.Nombre(), err)
	}
	if hab := h.habitacionDeLocked(p); hab != nil {
		return hab.Liberar(p)
	}
	return nil
}

// PacientesEnPasillo devuelve, en orden de admisión, los pacientes dormidos
// en el pasillo, para mandarles un camillero. El slice es nuevo: modificarlo
// no altera la lista interna del hospital.
func (h *Hospital) PacientesEnPasillo() []*Paciente {
	h.mu.Lock()
	defer h.mu.Unlock()
	var enPasillo []*Paciente
	for _, p := range h.pacientes {
		if p.Estado() == DormidoEnPasillo {
			enPasillo = append(enPasillo, p)
		}
	}
	return enPasillo
}

// ReporteSeveros cruza dos fuentes: los pacientes de nivel Severo y los
// episodios del historial ocurridos hoy. Devuelve cuántos episodios tuvo
// cada paciente Severo en el día; los que no tuvieron aparecen con 0.
func (h *Hospital) ReporteSeveros() map[*Paciente]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	reporte := make(map[*Paciente]int)
	for _, p := range h.pacientes {
		if p.Nivel() == Severo {
			reporte[p] = 0
		}
	}
	hoy := h.ahora()
	for _, r := range h.historial {
		if _, esSevero := reporte[r.paciente]; esSevero && mismoDia(r.fechaHora, hoy) {
			reporte[r.paciente]++
		}
	}
	return reporte
}

// Pacientes devuelve una copia de la lista de pacientes admitidos.
func (h *Hospital) Pacientes() []*Paciente {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.pacientes)
}

// Doctores devuelve una copia de la lista de doctores.
func (h *Hospital) Doctores() []*Doctor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.doctores)
}

// Personal devuelve una copia de la lista de todo el personal.
func (h *Hospital) Personal() []Atendedor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.personal)
}

// Habitaciones devuelve una copia de la lista de habitaciones.
func (h *Hospital) Habitaciones() []*Habitacion {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.habitaciones)
}

// Historial devuelve copias de todos los episodios, en orden de registro.
func (h *Hospital) Historial() []RegistroEpisodio {
	h.mu.Lock()
	defer h.mu.Unlock()
	historial := make([]RegistroEpisodio, 0, len(h.historial))
	for _, r := range h.historial {
		historial = append(historial, *r)
	}
	return historial
}

// Los métodos con sufijo Locked asumen que quien los llama ya tiene h.mu.
// Existen para que un método con lock pueda reutilizar lógica de otro sin
// volver a bloquear el mutex (sync.Mutex no es reentrante).

// contratarPersonalLocked agrega un Atendedor al personal.
func (h *Hospital) contratarPersonalLocked(a Atendedor) error {
	if a == nil {
		return ErrPersonalNulo
	}
	for _, b := range h.personal {
		if b.ID() == a.ID() {
			return fmt.Errorf("contratar %s: %w", a.ID(), ErrPersonalDuplicado)
		}
	}
	h.personal = append(h.personal, a)
	return nil
}

// asignarHabitacionLocked contiene la lógica de AsignarHabitacion.
func (h *Hospital) asignarHabitacionLocked(p *Paciente) (*Habitacion, error) {
	if !h.admitidoLocked(p) {
		return nil, fmt.Errorf("asignar habitación: %w", ErrPacienteNoAdmitido)
	}
	switch p.Estado() {
	case Despierto:
		return nil, fmt.Errorf("asignar habitación a %s: %w", p.Nombre(), ErrPacienteDespierto)
	case DormidoEnCama:
		return nil, fmt.Errorf("asignar habitación a %s: %w", p.Nombre(), ErrPacienteYaEnCama)
	}
	for _, hab := range h.habitaciones {
		if !hab.EstaDisponible() {
			continue
		}
		if err := hab.Ocupar(p); err != nil {
			return nil, fmt.Errorf("asignar habitación a %s: %w", p.Nombre(), err)
		}
		p.acostarEnCama(hab)
		if r := h.ultimoEpisodioLocked(p); r != nil && r.habitacion == nil {
			r.habitacion = hab
		}
		return hab, nil
	}
	return nil, fmt.Errorf("asignar habitación a %s: %w", p.Nombre(), ErrSinHabitacion)
}

// siguienteAtendedorLocked recorre el personal en orden circular a partir
// del turno actual y devuelve el primero disponible. Solo usa métodos de la
// interfaz.
func (h *Hospital) siguienteAtendedorLocked() (Atendedor, error) {
	n := len(h.personal)
	for i := 0; i < n; i++ {
		idx := (h.turno + i) % n
		if a := h.personal[idx]; a.Disponible() {
			h.turno = (idx + 1) % n
			return a, nil
		}
	}
	return nil, ErrSinPersonal
}

// admitidoLocked indica si el paciente pertenece a este hospital.
func (h *Hospital) admitidoLocked(p *Paciente) bool {
	return p != nil && slices.Contains(h.pacientes, p)
}

// habitacionDeLocked devuelve la habitación donde está el paciente, o nil.
func (h *Hospital) habitacionDeLocked(p *Paciente) *Habitacion {
	for _, hab := range h.habitaciones {
		if hab.tieneOcupante(p) {
			return hab
		}
	}
	return nil
}

// ultimoEpisodioLocked devuelve el episodio más reciente del paciente, o nil.
func (h *Hospital) ultimoEpisodioLocked(p *Paciente) *RegistroEpisodio {
	for i := len(h.historial) - 1; i >= 0; i-- {
		if h.historial[i].paciente == p {
			return h.historial[i]
		}
	}
	return nil
}

// mismoDia indica si dos instantes caen en la misma fecha de calendario.
func mismoDia(a, b time.Time) bool {
	a1, m1, d1 := a.Date()
	a2, m2, d2 := b.Date()
	return a1 == a2 && m1 == m2 && d1 == d2
}
