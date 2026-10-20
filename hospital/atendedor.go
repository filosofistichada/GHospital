package hospital

// Atendedor es cualquier miembro del personal que puede atender a un paciente
// que se quedó dormido. El hospital guarda su personal como []Atendedor y lo
// despacha sin saber si es un Doctor, un Camillero u otro tipo.
type Atendedor interface {
	ID() string
	Nombre() string
	// Disponible indica si puede recibir un paciente más en este momento.
	Disponible() bool
	// Atender se encarga del paciente dormido y devuelve el registro del
	// episodio, todavía sin ID ni habitación (eso lo completa el hospital).
	Atender(p *Paciente, ubicacion string) (RegistroEpisodio, error)
}

// llevaHistorial es una capacidad opcional del personal: guardar los
// episodios en los que intervino. El hospital no pregunta "¿eres Doctor?",
// sino "¿llevas historial?" (type assertion a esta interfaz). Así comparte
// con el atendedor el mismo *RegistroEpisodio y, si después se le asigna
// habitación al paciente, el atendedor ve el cambio.
type llevaHistorial interface {
	anotarEpisodio(r *RegistroEpisodio)
}

// Comprobación en tiempo de compilación: si Doctor o Camillero dejan de
// cumplir la interfaz, el programa no compila.
var (
	_ Atendedor      = (*Doctor)(nil)
	_ Atendedor      = (*Camillero)(nil)
	_ llevaHistorial = (*Doctor)(nil)
)
