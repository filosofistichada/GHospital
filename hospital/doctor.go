package hospital

import (
	"fmt"
	"slices"
)

// cupoPacientesPorDoctor es cuántos pacientes puede tener un doctor a cargo
// antes de dejar de estar disponible para el despacho.
const cupoPacientesPorDoctor = 3

// Doctor es un médico del hospital. Embebe Persona por valor e implementa
// Atendedor; además diagnostica y lleva historial de sus episodios.
//
// Sus métodos no son seguros para uso concurrente por sí solos: el Hospital
// los llama con su mutex tomado. No llames MisEpisodios ni Pacientes
// mientras otras goroutines registran episodios; hazlo cuando terminen.
type Doctor struct {
	Persona
	especialidad string
	pacientes    []*Paciente
	episodios    []*RegistroEpisodio
}

// NuevoDoctor crea un doctor sin pacientes ni episodios.
func NuevoDoctor(id, nombre string, edad int, especialidad string) *Doctor {
	return &Doctor{Persona: NuevaPersona(id, nombre, edad), especialidad: especialidad}
}

// Especialidad devuelve la especialidad del doctor.
func (d *Doctor) Especialidad() string { return d.especialidad }

// Pacientes devuelve una copia de la lista de pacientes a cargo.
func (d *Doctor) Pacientes() []*Paciente { return slices.Clone(d.pacientes) }

// DiagnosticarPaciente deja al paciente a cargo de este doctor. Si lo tenía
// otro doctor, se lo quita. Falla si el doctor ya no tiene cupo.
func (d *Doctor) DiagnosticarPaciente(p *Paciente) error {
	if p == nil {
		return ErrPacienteNulo
	}
	if p.doctorAsignado == d {
		return nil
	}
	if len(d.pacientes) >= cupoPacientesPorDoctor {
		return fmt.Errorf("%s: %w", d.Nombre(), ErrDoctorSinCupo)
	}
	if anterior := p.doctorAsignado; anterior != nil {
		anterior.quitarPaciente(p)
	}
	d.pacientes = append(d.pacientes, p)
	p.doctorAsignado = d
	return nil
}

// AtenderEmergenciaDeSueno revisa a un paciente dormido y lo diagnostica.
// Falla si el paciente está despierto.
func (d *Doctor) AtenderEmergenciaDeSueno(p *Paciente) error {
	if p == nil {
		return ErrPacienteNulo
	}
	if p.Estado() == Despierto {
		return fmt.Errorf("%s: %w", p.Nombre(), ErrPacienteDespierto)
	}
	return d.DiagnosticarPaciente(p)
}

// Disponible indica si el doctor todavía tiene cupo para otro paciente.
func (d *Doctor) Disponible() bool {
	return len(d.pacientes) < cupoPacientesPorDoctor
}

// Atender implementa Atendedor: el doctor atiende la emergencia (lo que
// incluye diagnosticar) y devuelve el registro del episodio.
func (d *Doctor) Atender(p *Paciente, ubicacion string) (RegistroEpisodio, error) {
	if err := d.AtenderEmergenciaDeSueno(p); err != nil {
		return RegistroEpisodio{}, err
	}
	return nuevoRegistro(p, d, ubicacion), nil
}

// MisEpisodios devuelve los episodios en los que intervino el doctor. Son
// copias: modificar el resultado no altera el historial del doctor.
func (d *Doctor) MisEpisodios() []RegistroEpisodio {
	episodios := make([]RegistroEpisodio, 0, len(d.episodios))
	for _, r := range d.episodios {
		episodios = append(episodios, *r)
	}
	return episodios
}

// anotarEpisodio implementa llevaHistorial: guarda el mismo puntero que el
// hospital, para ver los cambios posteriores (como la habitación asignada).
func (d *Doctor) anotarEpisodio(r *RegistroEpisodio) {
	d.episodios = append(d.episodios, r)
}

// quitarPaciente saca a un paciente de la lista de este doctor.
func (d *Doctor) quitarPaciente(p *Paciente) {
	if i := slices.Index(d.pacientes, p); i >= 0 {
		d.pacientes = slices.Delete(d.pacientes, i, i+1)
	}
}
