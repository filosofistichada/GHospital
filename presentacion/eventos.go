package presentacion

import (
	"fmt"
	"strings"

	"costenos-narcolepsia/simulacion"
)

// LineaEvento da formato a un evento de la simulación en una línea, con el
// color del equipo del paciente.
func (c *Consola) LineaEvento(ev simulacion.Evento) string {
	quien := ev.Paciente.Nombre()
	var texto string
	switch ev.Tipo {
	case simulacion.SeDurmioEnCama:
		texto = fmt.Sprintf("zZz %s se durmió %s -> consiguió cama", quien, ev.Ubicacion)
	case simulacion.SeDurmioEnPasillo:
		texto = fmt.Sprintf("zZz %s se durmió %s -> al pasillo, no hay cama", quien, ev.Ubicacion)
	case simulacion.LlevadoACama:
		texto = fmt.Sprintf("[=] Ronda del pasillo: %s -> habitación %d", quien, ev.Habitacion)
	case simulacion.Desperto:
		texto = fmt.Sprintf("o_o %s se despertó", quien)
	default:
		texto = fmt.Sprintf("!!! %s: %v", quien, ev.Err)
	}
	return c.pintar(c.equipoDe(ev.Paciente).ansi, fmt.Sprintf("[%5.2fs] %s", ev.Momento.Seconds(), texto))
}

// ResumenSimulacion da formato a los totales de la simulación.
func (c *Consola) ResumenSimulacion(r simulacion.Resumen) string {
	lineas := []string{
		fmt.Sprintf("  Ataques de sueño:            %d", r.Ataques),
		fmt.Sprintf("    con cama de una vez:       %d", r.ConCama),
		fmt.Sprintf("    al pasillo sin cama:       %d", r.EnPasillo),
		fmt.Sprintf("  Llevados a cama por la ronda: %d", r.LlevadosPorRonda),
		fmt.Sprintf("  Despertares:                 %d", r.Despertares),
	}
	if r.Fallos > 0 {
		lineas = append(lineas, c.pintar(ansiRojo, fmt.Sprintf("  Fallos inesperados:          %d", r.Fallos)))
	}
	return strings.Join(lineas, "\n")
}
