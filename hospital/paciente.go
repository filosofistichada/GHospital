package hospital

import "fmt"

// NivelNarcolepsia indica qué tan fuerte es la narcolepsia de un paciente.
type NivelNarcolepsia int

// Niveles de narcolepsia, de menor a mayor.
const (
	Leve NivelNarcolepsia = iota
	Moderado
	Severo
)

// String devuelve el nivel en texto legible.
func (n NivelNarcolepsia) String() string {
	switch n {
	case Leve:
		return "leve"
	case Moderado:
		return "moderado"
	case Severo:
		return "severo"
	default:
		return "desconocido"
	}
}

// EstadoPaciente indica si el paciente está despierto o dónde se quedó dormido.
type EstadoPaciente int

// Estados posibles de un paciente. El valor cero es Despierto, así que un
// paciente recién creado empieza despierto.
const (
	Despierto EstadoPaciente = iota
	DormidoEnPasillo
	DormidoEnCama
)

// String devuelve el estado en texto legible.
func (e EstadoPaciente) String() string {
	switch e {
	case Despierto:
		return "despierto"
	case DormidoEnPasillo:
		return "dormido en el pasillo"
	case DormidoEnCama:
		return "dormido en cama"
	default:
		return "desconocido"
	}
}

// Paciente es un costeño con narcolepsia admitido en el hospital.
// Embebe Persona por valor para reutilizar su identidad.
//
// Los métodos de Paciente no son seguros para uso concurrente por sí solos:
// los cambios de estado deben pasar por el Hospital.
type Paciente struct {
	Persona
	nivel           NivelNarcolepsia
	estado          EstadoPaciente
	ubicacionActual string
	doctorAsignado  *Doctor
}

// NuevoPaciente crea un paciente despierto en la sala de espera.
func NuevoPaciente(id, nombre string, edad int, nivel NivelNarcolepsia) *Paciente {
	return &Paciente{
		Persona:         NuevaPersona(id, nombre, edad),
		nivel:           nivel,
		estado:          Despierto,
		ubicacionActual: "sala de espera",
	}
}

// Nivel devuelve el nivel de narcolepsia del paciente.
func (p *Paciente) Nivel() NivelNarcolepsia { return p.nivel }

// Estado devuelve si el paciente está despierto o dónde está dormido.
func (p *Paciente) Estado() EstadoPaciente { return p.estado }

// Ubicacion devuelve el último lugar conocido del paciente.
func (p *Paciente) Ubicacion() string { return p.ubicacionActual }

// DoctorAsignado devuelve el doctor que diagnosticó al paciente, o nil si
// todavía no lo ha visto ningún doctor.
func (p *Paciente) DoctorAsignado() *Doctor { return p.doctorAsignado }

// SufrirAtaqueDeSueno deja al paciente dormido en el pasillo, en la ubicación
// donde le dio el ataque. Falla si ya estaba dormido.
func (p *Paciente) SufrirAtaqueDeSueno(ubicacion string) error {
	if p.estado != Despierto {
		return ErrPacienteDormido
	}
	p.estado = DormidoEnPasillo
	p.ubicacionActual = ubicacion
	return nil
}

// Despertar pone al paciente despierto. Falla si ya estaba despierto.
// No libera la habitación: eso lo hace Hospital.DespertarPaciente.
func (p *Paciente) Despertar() error {
	if p.estado == Despierto {
		return ErrPacienteDespierto
	}
	p.estado = Despierto
	return nil
}

// acostarEnCama marca al paciente como dormido en la habitación dada.
// Es interna: solo el hospital decide quién recibe cama.
func (p *Paciente) acostarEnCama(h *Habitacion) {
	p.estado = DormidoEnCama
	p.ubicacionActual = fmt.Sprintf("habitación %d", h.Numero())
}
