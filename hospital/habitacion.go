package hospital

import (
	"fmt"
	"slices"
)

// EstadoHabitacion indica si una habitación todavía recibe pacientes.
type EstadoHabitacion int

// Estados posibles de una habitación.
const (
	Disponible EstadoHabitacion = iota
	Ocupada
)

// String devuelve el estado en texto legible.
func (e EstadoHabitacion) String() string {
	switch e {
	case Disponible:
		return "disponible"
	case Ocupada:
		return "ocupada"
	default:
		return "desconocido"
	}
}

// Habitacion es un cuarto del hospital con una capacidad fija de camas.
// Solo lleva la cuenta de sus ocupantes; el estado del paciente lo cambia
// el Hospital. Sus métodos no son seguros para uso concurrente por sí solos.
type Habitacion struct {
	numero    int
	capacidad int
	estado    EstadoHabitacion
	ocupantes []*Paciente
}

// NuevaHabitacion crea una habitación disponible. Falla si la capacidad es
// menor que 1.
func NuevaHabitacion(numero, capacidad int) (*Habitacion, error) {
	if capacidad < 1 {
		return nil, fmt.Errorf("habitación %d: %w", numero, ErrCapacidadInvalida)
	}
	return &Habitacion{numero: numero, capacidad: capacidad, estado: Disponible}, nil
}

// Numero devuelve el número de la habitación.
func (h *Habitacion) Numero() int { return h.numero }

// Capacidad devuelve cuántos pacientes caben.
func (h *Habitacion) Capacidad() int { return h.capacidad }

// Estado devuelve si la habitación está disponible u ocupada.
func (h *Habitacion) Estado() EstadoHabitacion { return h.estado }

// Ocupantes devuelve una copia de la lista de pacientes en la habitación,
// para que nadie de afuera pueda modificar la lista interna.
func (h *Habitacion) Ocupantes() []*Paciente { return slices.Clone(h.ocupantes) }

// EstaDisponible indica si queda al menos una cama libre.
func (h *Habitacion) EstaDisponible() bool { return h.estado == Disponible }

// Ocupar acuesta al paciente en la habitación. Si con él se llena, pasa a
// Ocupada.
func (h *Habitacion) Ocupar(p *Paciente) error {
	if p == nil {
		return ErrPacienteNulo
	}
	if h.tieneOcupante(p) {
		return fmt.Errorf("habitación %d: %w", h.numero, ErrPacienteYaEnCama)
	}
	if !h.EstaDisponible() {
		return fmt.Errorf("habitación %d: %w", h.numero, ErrHabitacionLlena)
	}
	h.ocupantes = append(h.ocupantes, p)
	if len(h.ocupantes) == h.capacidad {
		h.estado = Ocupada
	}
	return nil
}

// Liberar saca al paciente de la habitación, que vuelve a quedar Disponible.
func (h *Habitacion) Liberar(p *Paciente) error {
	i := slices.Index(h.ocupantes, p)
	if i < 0 {
		return fmt.Errorf("habitación %d: %w", h.numero, ErrNoEsOcupante)
	}
	h.ocupantes = slices.Delete(h.ocupantes, i, i+1)
	h.estado = Disponible
	return nil
}

// tieneOcupante indica si el paciente está acostado en esta habitación.
func (h *Habitacion) tieneOcupante(p *Paciente) bool {
	return slices.Contains(h.ocupantes, p)
}
