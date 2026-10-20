package hospital

// Persona agrupa los datos de identidad que comparten pacientes, doctores y
// camilleros. Se embebe por valor: cada entidad tiene su propia copia de la
// identidad y hereda (por promoción) los métodos ID, Nombre y Edad.
type Persona struct {
	id     string
	nombre string
	edad   int
}

// NuevaPersona crea una Persona con sus datos de identidad.
func NuevaPersona(id, nombre string, edad int) Persona {
	return Persona{id: id, nombre: nombre, edad: edad}
}

// ID devuelve el identificador único de la persona.
func (p Persona) ID() string { return p.id }

// Nombre devuelve el nombre completo de la persona.
func (p Persona) Nombre() string { return p.nombre }

// Edad devuelve la edad en años.
func (p Persona) Edad() int { return p.edad }
