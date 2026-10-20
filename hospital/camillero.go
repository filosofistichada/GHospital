package hospital

import "fmt"

// Camillero lleva pacientes dormidos a una cama, pero no diagnostica: al
// atender, el paciente queda sin doctor asignado. Embebe Persona, así que
// ID y Nombre de la interfaz Atendedor le vienen promovidos.
type Camillero struct {
	Persona
}

// NuevoCamillero crea un camillero.
func NuevoCamillero(id, nombre string, edad int) *Camillero {
	return &Camillero{Persona: NuevaPersona(id, nombre, edad)}
}

// Disponible siempre es verdadero: al camillero le dicen "Piernas" por algo.
func (c *Camillero) Disponible() bool { return true }

// Atender implementa Atendedor: recoge al paciente dormido y devuelve el
// registro del episodio, sin diagnosticarlo.
func (c *Camillero) Atender(p *Paciente, ubicacion string) (RegistroEpisodio, error) {
	if p == nil {
		return RegistroEpisodio{}, ErrPacienteNulo
	}
	if p.Estado() == Despierto {
		return RegistroEpisodio{}, fmt.Errorf("%s: %w", p.Nombre(), ErrPacienteDespierto)
	}
	return nuevoRegistro(p, c, ubicacion), nil
}
