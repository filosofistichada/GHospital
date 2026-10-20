package presentacion

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"costenos-narcolepsia/hospital"
)

// anchoCarta es el ancho del texto dentro de cada carta, sin contar bordes.
const anchoCarta = 32

// Códigos ANSI de formato.
const (
	ansiReset   = "\033[0m"
	ansiNegrita = "\033[1m"
	ansiRojo    = "\033[91m"
	ansiTenue   = "\033[2m"
)

// Equipo es el equipo de la costa del que es hincha un paciente. Define el
// color de sus cartas. Vive aquí y no en hospital porque es apariencia: al
// hospital no le importa de qué equipo eres.
type Equipo struct {
	nombre string
	apodo  string
	color  string
	ansi   string
}

// Equipos disponibles. Seleccion es el comodín multicolor.
var (
	Junior         = Equipo{"Junior", "el Tiburón", "rojo", "\033[91m"}
	UnionMagdalena = Equipo{"Unión Magdalena", "el Ciclón bananero", "azul", "\033[94m"}
	RealCartagena  = Equipo{"Real Cartagena", "el Heroico", "verde", "\033[92m"}
	Jaguares       = Equipo{"Jaguares", "el Jaguar", "amarillo", "\033[93m"}
	Seleccion      = Equipo{"Selección", "en el Metropolitano", "multicolor", "\033[95m"}
)

// Nombre devuelve el nombre del equipo.
func (e Equipo) Nombre() string { return e.nombre }

// Apodo devuelve el apodo del equipo.
func (e Equipo) Apodo() string { return e.apodo }

// Color devuelve el color del equipo en texto.
func (e Equipo) Color() string { return e.color }

// Consola da formato de texto al modelo del hospital: cartas, títulos y
// mensajes. Solo formatea; nunca cambia el estado del hospital.
type Consola struct {
	conColor bool
	equipos  map[string]Equipo // ID del paciente -> equipo
}

// NuevaConsola crea una consola. Si conColor es falso, no se emiten códigos
// ANSI. Los pacientes sin equipo en el mapa salen con el comodín Seleccion.
func NuevaConsola(conColor bool, equipos map[string]Equipo) *Consola {
	return &Consola{conColor: conColor, equipos: equipos}
}

// Titulo da formato al encabezado de una sección.
func (c *Consola) Titulo(texto string) string {
	return "\n" + c.pintar(ansiNegrita, "== "+texto+" ==")
}

// Nota da formato a un comentario secundario (chistes, aclaraciones).
func (c *Consola) Nota(texto string) string {
	return c.pintar(ansiTenue, strings.Join(envolver(texto, 3*(anchoCarta+4)), "\n"))
}

// Error da formato a un error para que se lea claro en la salida.
func (c *Consola) Error(err error) string {
	return c.pintar(ansiRojo, "✗ "+strings.Join(envolver(err.Error(), 3*(anchoCarta+4)-2), "\n  "))
}

// CartaPaciente dibuja la carta de un paciente con el color de su equipo.
func (c *Consola) CartaPaciente(p *hospital.Paciente) string {
	eq := c.equipoDe(p)
	lineas := []string{izqDer(p.ID(), strings.ToUpper(eq.color+" · "+eq.nombre), anchoCarta)}
	lineas = append(lineas, envolver(p.Nombre(), anchoCarta)...)
	lineas = append(lineas,
		fmt.Sprintf("%d años · nivel %s", p.Edad(), p.Nivel()),
		iconoEstado(p.Estado())+" "+p.Estado().String(),
	)
	lugar := "lugar: "
	if p.Estado() == hospital.Despierto {
		lugar = "último lugar: "
	}
	lineas = append(lineas, envolver(lugar+p.Ubicacion(), anchoCarta)...)
	doctor := "sin diagnóstico"
	if d := p.DoctorAsignado(); d != nil {
		doctor = d.Nombre()
	}
	lineas = append(lineas, envolver("doctor: "+doctor, anchoCarta)...)
	return c.carta(lineas, eq.ansi)
}

// CartaEpisodio dibuja la carta de un episodio con el color del equipo del
// paciente que se durmió.
func (c *Consola) CartaEpisodio(r hospital.RegistroEpisodio) string {
	p := r.Paciente()
	lineas := []string{izqDer(r.ID(), r.FechaHora().Format("2006-01-02 15:04"), anchoCarta)}
	lineas = append(lineas, envolver(p.ID()+" "+p.Nombre(), anchoCarta)...)
	lineas = append(lineas, envolver("se durmió "+r.Ubicacion(), anchoCarta)...)
	lineas = append(lineas, envolver("atención: "+r.Atendedor().Nombre(), anchoCarta)...)
	destino := "-> se quedó en el pasillo"
	if hab := r.Habitacion(); hab != nil {
		destino = fmt.Sprintf("-> habitación %d", hab.Numero())
	}
	lineas = append(lineas, destino)
	return c.carta(lineas, c.equipoDe(p).ansi)
}

// CartaHabitacion dibuja una habitación con sus ocupantes.
func (c *Consola) CartaHabitacion(h *hospital.Habitacion) string {
	lineas := []string{
		izqDer(fmt.Sprintf("Habitación %d", h.Numero()), h.Estado().String(), anchoCarta),
		fmt.Sprintf("capacidad: %d", h.Capacidad()),
	}
	ocupantes := h.Ocupantes()
	if len(ocupantes) == 0 {
		lineas = append(lineas, "(cama libre)")
	}
	for _, p := range ocupantes {
		lineas = append(lineas, envolver("roncando: "+p.Nombre(), anchoCarta)...)
	}
	color := ""
	if len(ocupantes) > 0 {
		color = c.equipoDe(ocupantes[0]).ansi
	}
	return c.carta(lineas, color)
}

// ReporteSeveros da formato al reporte de severos, ordenado por ID para que
// la salida sea siempre la misma (un map no tiene orden).
func (c *Consola) ReporteSeveros(reporte map[*hospital.Paciente]int) string {
	pacientes := make([]*hospital.Paciente, 0, len(reporte))
	ancho := 0
	for p := range reporte {
		pacientes = append(pacientes, p)
		ancho = max(ancho, utf8.RuneCountInString(p.Nombre()))
	}
	slices.SortFunc(pacientes, func(a, b *hospital.Paciente) int {
		return strings.Compare(a.ID(), b.ID())
	})
	var b strings.Builder
	for _, p := range pacientes {
		n := reporte[p]
		palabra := "episodios"
		if n == 1 {
			palabra = "episodio"
		}
		linea := fmt.Sprintf("%s  %s  %d %s hoy", p.ID(), rellenar(p.Nombre(), ancho), n, palabra)
		b.WriteString(c.pintar(c.equipoDe(p).ansi, linea) + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// EnFila pone las cartas una al lado de la otra, de a porFila por fila.
func EnFila(cartas []string, porFila int) string {
	var filas []string
	for inicio := 0; inicio < len(cartas); inicio += porFila {
		grupo := cartas[inicio:min(inicio+porFila, len(cartas))]
		filas = append(filas, unirLado(grupo, anchoCarta))
	}
	return strings.Join(filas, "\n")
}

// ChisteBarranquilla arma el chiste del paso en que un paciente despierta y
// otro se queda con su habitación.
func ChisteBarranquilla(despierto, otro string, numero int) string {
	return fmt.Sprintf("¡El que se fue pa' Barranquilla perdió su silla! %s se despertó para ir a ver al Junior y %s ya está roncando en la habitación %d.",
		despierto, otro, numero)
}

// carta dibuja un recuadro del ancho estándar con las líneas dadas, todo en
// el color indicado.
func (c *Consola) carta(lineas []string, ansi string) string {
	return c.cartaDeAncho(lineas, ansi, anchoCarta)
}

// cartaDeAncho dibuja un recuadro con texto de ancho caracteres por línea.
func (c *Consola) cartaDeAncho(lineas []string, ansi string, ancho int) string {
	borde := strings.Repeat("─", ancho+2)
	salida := []string{c.pintar(ansi, "╭"+borde+"╮")}
	for _, l := range lineas {
		salida = append(salida, c.pintar(ansi, "│ "+rellenar(l, ancho)+" │"))
	}
	salida = append(salida, c.pintar(ansi, "╰"+borde+"╯"))
	return strings.Join(salida, "\n")
}

// pintar envuelve el texto en un código ANSI si los colores están activos.
func (c *Consola) pintar(ansi, texto string) string {
	if !c.conColor || ansi == "" {
		return texto
	}
	return ansi + texto + ansiReset
}

// equipoDe devuelve el equipo del paciente, o el comodín si no tiene.
func (c *Consola) equipoDe(p *hospital.Paciente) Equipo {
	if eq, ok := c.equipos[p.ID()]; ok {
		return eq
	}
	return Seleccion
}

// iconoEstado dibuja la cara del paciente según su estado.
func iconoEstado(e hospital.EstadoPaciente) string {
	switch e {
	case hospital.DormidoEnPasillo:
		return "(-_-) zZz"
	case hospital.DormidoEnCama:
		return "(-_-) zZz [cama]"
	default:
		return "(o_o)"
	}
}

// unirLado pega cartas de igual ancho de texto una al lado de la otra; las
// más bajas se completan con espacios.
func unirLado(cartas []string, ancho int) string {
	partidas := make([][]string, len(cartas))
	alto := 0
	for i, c := range cartas {
		partidas[i] = strings.Split(c, "\n")
		alto = max(alto, len(partidas[i]))
	}
	vacio := strings.Repeat(" ", ancho+4) // texto + bordes + márgenes
	filas := make([]string, alto)
	for f := range filas {
		piezas := make([]string, len(partidas))
		for i, lineas := range partidas {
			piezas[i] = vacio
			if f < len(lineas) {
				piezas[i] = lineas[f]
			}
		}
		filas[f] = strings.Join(piezas, "  ")
	}
	return strings.Join(filas, "\n")
}

// envolver parte el texto en líneas de como mucho ancho caracteres, sin
// cortar palabras (salvo que una sola palabra sea más larga que el ancho).
func envolver(texto string, ancho int) []string {
	var lineas []string
	actual := ""
	for _, palabra := range strings.Fields(texto) {
		for utf8.RuneCountInString(palabra) > ancho {
			if actual != "" {
				lineas = append(lineas, actual)
				actual = ""
			}
			r := []rune(palabra)
			lineas = append(lineas, string(r[:ancho]))
			palabra = string(r[ancho:])
		}
		switch {
		case actual == "":
			actual = palabra
		case utf8.RuneCountInString(actual)+1+utf8.RuneCountInString(palabra) <= ancho:
			actual += " " + palabra
		default:
			lineas = append(lineas, actual)
			actual = palabra
		}
	}
	if actual != "" || len(lineas) == 0 {
		lineas = append(lineas, actual)
	}
	return lineas
}

// izqDer pone un texto a la izquierda y otro a la derecha en el ancho dado.
func izqDer(izq, der string, ancho int) string {
	espacio := ancho - utf8.RuneCountInString(izq) - utf8.RuneCountInString(der)
	if espacio < 1 {
		return rellenar(izq, ancho)
	}
	return izq + strings.Repeat(" ", espacio) + der
}

// rellenar completa con espacios hasta el ancho, contando runas (no bytes)
// para que las tildes no descuadren las cartas.
func rellenar(texto string, ancho int) string {
	n := utf8.RuneCountInString(texto)
	if n >= ancho {
		return texto
	}
	return texto + strings.Repeat(" ", ancho-n)
}
