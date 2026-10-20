package hospital

import (
	"fmt"
	"time"
)

// RegistroEpisodio documenta un ataque de sueño: quién se durmió, dónde,
// cuándo, quién lo atendió y a qué habitación fue a parar (si consiguió).
type RegistroEpisodio struct {
	id         string
	fechaHora  time.Time
	paciente   *Paciente
	atendedor  Atendedor
	ubicacion  string
	habitacion *Habitacion
}

// nuevoRegistro crea un registro con la hora actual, sin ID ni habitación.
func nuevoRegistro(p *Paciente, a Atendedor, ubicacion string) RegistroEpisodio {
	return RegistroEpisodio{
		fechaHora: time.Now(),
		paciente:  p,
		atendedor: a,
		ubicacion: ubicacion,
	}
}

// ID devuelve el identificador del episodio (por ejemplo "E-001").
func (r RegistroEpisodio) ID() string { return r.id }

// FechaHora devuelve cuándo ocurrió el ataque.
func (r RegistroEpisodio) FechaHora() time.Time { return r.fechaHora }

// Paciente devuelve el paciente que se durmió.
func (r RegistroEpisodio) Paciente() *Paciente { return r.paciente }

// Atendedor devuelve quién atendió el episodio.
func (r RegistroEpisodio) Atendedor() Atendedor { return r.atendedor }

// Ubicacion devuelve dónde se quedó dormido el paciente.
func (r RegistroEpisodio) Ubicacion() string { return r.ubicacion }

// Habitacion devuelve la habitación asignada, o nil si el paciente se quedó
// en el pasillo.
func (r RegistroEpisodio) Habitacion() *Habitacion { return r.habitacion }

// Resumen describe el episodio en una línea.
func (r RegistroEpisodio) Resumen() string {
	destino := "se quedó en el pasillo"
	if r.habitacion != nil {
		destino = fmt.Sprintf("-> habitación %d", r.habitacion.Numero())
	}
	quien := "nadie"
	if r.atendedor != nil {
		quien = r.atendedor.Nombre()
	}
	return fmt.Sprintf("%s %s %s %s | %s | atención: %s | %s",
		r.id, r.fechaHora.Format("2006-01-02 15:04"),
		r.paciente.ID(), r.paciente.Nombre(), r.ubicacion, quien, destino)
}
