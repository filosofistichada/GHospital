package virus

import "fmt"

// Comprobación en tiempo de compilación de que cada tipo implementa Carta.
var (
	_ Carta = (*CartaPaciente)(nil)
	_ Carta = (*CartaAtaque)(nil)
	_ Carta = (*CartaCama)(nil)
	_ Carta = (*CartaDoctor)(nil)
	_ Carta = (*CartaCamillero)(nil)
	_ Carta = (*CartaCoste)(nil)
	_ Carta = (*CartaCambioDeNovia)(nil)
	_ Carta = (*CartaLaMoza)(nil)
	_ Carta = (*CartaChisme)(nil)
	_ Carta = (*CartaPico)(nil)
	_ Carta = (*CartaCachos)(nil)
)

// ---------- Paciente ----------

// CartaPaciente se pone en tu tablero. No puedes tener dos del mismo color.
type CartaPaciente struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaPaciente) Tipo() TipoCarta { return TipoPaciente }

// Requiere implementa Carta: no necesita objetivo.
func (c *CartaPaciente) Requiere() Requisitos { return Requisitos{} }

// Jugar pone el paciente en tu tablero.
func (c *CartaPaciente) Jugar(j *Juego, _ Objetivo) error {
	yo := j.actual()
	if _, ya := yo.tablero[c.color]; ya {
		return fmt.Errorf("%s: %w", c.color.Equipo(), ErrColorRepetido)
	}
	yo.tablero[c.color] = &PacienteEnJuego{carta: c}
	return nil
}

// ---------- Ataque de sueño ----------

// CartaAtaque (una comida típica) duerme a un paciente rival de su color.
// Si ya estaba dormido, sale del tablero; si tenía una cama, se anulan.
type CartaAtaque struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaAtaque) Tipo() TipoCarta { return TipoAtaque }

// Requiere implementa Carta: un paciente de un rival.
func (c *CartaAtaque) Requiere() Requisitos { return Requisitos{Rival: true, PacienteRival: true} }

// Jugar ataca al paciente del rival.
func (c *CartaAtaque) Jugar(j *Juego, obj Objetivo) error {
	rival, p, err := j.pacienteDeRival(obj)
	if err != nil {
		return err
	}
	if !c.color.combinaCon(p.Color()) {
		return ErrColorIncompatible
	}
	if p.Estable() {
		return ErrPacienteEstable
	}
	j.atacar(rival, p, c)
	return nil
}

// ---------- Camas: cama normal, doctor y camillero ----------

// CartaCama (un remedio) va sobre tu paciente de su color: si estaba
// dormido lo despierta; con dos camas queda estable.
type CartaCama struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaCama) Tipo() TipoCarta { return TipoCama }

// Requiere implementa Carta: un paciente propio.
func (c *CartaCama) Requiere() Requisitos { return Requisitos{PacientePropio: true} }

// Jugar pone la cama.
func (c *CartaCama) Jugar(j *Juego, obj Objetivo) error { return jugarCama(j, obj, c) }

// CartaDoctor funciona como una cama de cualquier color y puede despertar a
// un dormido, porque sabe diagnosticar.
type CartaDoctor struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaDoctor) Tipo() TipoCarta { return TipoCama }

// Requiere implementa Carta: un paciente propio.
func (c *CartaDoctor) Requiere() Requisitos { return Requisitos{PacientePropio: true} }

// Jugar pone la cama del doctor.
func (c *CartaDoctor) Jugar(j *Juego, obj Objetivo) error { return jugarCama(j, obj, c) }

// CartaCamillero pone una cama de cualquier color, pero solo a un paciente
// despierto: no diagnostica (la misma regla que en el hospital).
type CartaCamillero struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaCamillero) Tipo() TipoCarta { return TipoCama }

// Requiere implementa Carta: un paciente propio.
func (c *CartaCamillero) Requiere() Requisitos { return Requisitos{PacientePropio: true} }

// Jugar pone la cama, salvo que el paciente esté dormido.
func (c *CartaCamillero) Jugar(j *Juego, obj Objetivo) error {
	if p, ok := j.actual().tablero[obj.PacientePropio]; ok && p.Dormido() {
		return ErrCamilleroNoDiagnost
	}
	return jugarCama(j, obj, c)
}

// jugarCama es la lógica común de las cartas de cama. Recibe la carta
// concreta para que sea esa (y no otra) la que quede encima del paciente.
func jugarCama(j *Juego, obj Objetivo, carta Carta) error {
	p, ok := j.actual().tablero[obj.PacientePropio]
	if !ok {
		return ErrSinPaciente
	}
	if !carta.Color().combinaCon(p.Color()) {
		return ErrColorIncompatible
	}
	if p.Estable() {
		return ErrPacienteEstable
	}
	if p.Dormido() {
		// La cama despierta al paciente: el ataque y la cama se anulan.
		j.descartar(p.ataques...)
		j.descartar(carta)
		p.ataques = nil
		return nil
	}
	p.camas = append(p.camas, carta)
	return nil
}

// ---------- El Coste, el Paciente Cero ----------

// frasesCoste se muestran al azar cuando juega el Coste.
var frasesCoste = []string{
	"No es lo que parece, mi amor, yo estaba dormido.",
	"Esa no es mi moza, es mi prima. Todas son mis primas.",
	"Yo no te puse los cachos, tú te los imaginaste porque tienes narcolepsia.",
	"El Coste dice que se fue a comprar hielo. Lleva tres meses comprando hielo.",
}

// CartaCoste pone un ataque a todos los pacientes de un rival a la vez,
// menos a los estables.
type CartaCoste struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaCoste) Tipo() TipoCarta { return TipoAtaque }

// Requiere implementa Carta: un rival.
func (c *CartaCoste) Requiere() Requisitos { return Requisitos{Rival: true} }

// Jugar ataca a todos los pacientes no estables del rival.
func (c *CartaCoste) Jugar(j *Juego, obj Objetivo) error {
	rival, err := j.rival(obj.Rival)
	if err != nil {
		return err
	}
	var victimas []*PacienteEnJuego
	for _, p := range rival.Pacientes() {
		if !p.Estable() {
			victimas = append(victimas, p)
		}
	}
	if len(victimas) == 0 {
		return ErrCosteSinVictimas
	}
	j.anunciar("Pasó el Coste por el tablero de %s: \"%s\"", rival.nombre, frasesCoste[j.azar.IntN(len(frasesCoste))])
	for _, p := range victimas {
		j.atacar(rival, p, nil)
	}
	j.descartar(c)
	return nil
}

// ---------- Tratamientos costeños ----------

// CartaCambioDeNovia ("Me cambié de novia") intercambia uno de tus pacientes
// con uno de un rival.
type CartaCambioDeNovia struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaCambioDeNovia) Tipo() TipoCarta { return TipoTratamiento }

// Requiere implementa Carta: tu paciente y uno del rival.
func (c *CartaCambioDeNovia) Requiere() Requisitos {
	return Requisitos{Rival: true, PacienteRival: true, PacientePropio: true}
}

// Jugar intercambia los dos pacientes con lo que tengan encima.
func (c *CartaCambioDeNovia) Jugar(j *Juego, obj Objetivo) error {
	rival, suyo, err := j.pacienteDeRival(obj)
	if err != nil {
		return err
	}
	yo := j.actual()
	mio, ok := yo.tablero[obj.PacientePropio]
	if !ok {
		return ErrSinPaciente
	}
	if mio.Estable() || suyo.Estable() {
		return ErrPacienteEstable
	}
	if mio.Color() != suyo.Color() {
		if _, ya := yo.tablero[suyo.Color()]; ya {
			return ErrColorRepetido
		}
		if _, ya := rival.tablero[mio.Color()]; ya {
			return ErrColorRepetido
		}
	}
	delete(yo.tablero, mio.Color())
	delete(rival.tablero, suyo.Color())
	yo.tablero[suyo.Color()] = suyo
	rival.tablero[mio.Color()] = mio
	j.anunciar("%s se cambió de novia con %s: ahora el hincha del %s es suyo. Dice que fue amor a primera vista.",
		yo.nombre, rival.nombre, suyo.Color().Equipo())
	j.descartar(c)
	return nil
}

// CartaLaMoza ("La moza") te roba un paciente de un rival.
type CartaLaMoza struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaLaMoza) Tipo() TipoCarta { return TipoTratamiento }

// Requiere implementa Carta: un paciente del rival.
func (c *CartaLaMoza) Requiere() Requisitos { return Requisitos{Rival: true, PacienteRival: true} }

// Jugar pasa el paciente del rival a tu tablero.
func (c *CartaLaMoza) Jugar(j *Juego, obj Objetivo) error {
	rival, suyo, err := j.pacienteDeRival(obj)
	if err != nil {
		return err
	}
	if suyo.Estable() {
		return ErrPacienteEstable
	}
	yo := j.actual()
	if _, ya := yo.tablero[suyo.Color()]; ya {
		return ErrColorRepetido
	}
	delete(rival.tablero, suyo.Color())
	yo.tablero[suyo.Color()] = suyo
	j.anunciar("La moza se llevó al hincha del %s de %s: llega sin avisar, se acomoda en tu cama y encima pide que le hagan el desayuno.",
		suyo.Color().Equipo(), rival.nombre)
	j.descartar(c)
	return nil
}

// CartaChisme ("Chisme de la vecina") pasa tus ataques a pacientes libres de
// los rivales que combinen en color.
type CartaChisme struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaChisme) Tipo() TipoCarta { return TipoTratamiento }

// Requiere implementa Carta: no necesita objetivo.
func (c *CartaChisme) Requiere() Requisitos { return Requisitos{} }

// Jugar reparte tus ataques entre los rivales.
func (c *CartaChisme) Jugar(j *Juego, _ Objetivo) error {
	type traspaso struct{ desde, hacia *PacienteEnJuego }
	var plan []traspaso
	ocupados := make(map[*PacienteEnJuego]bool)
	// Primero se planea todo; si no hay ningún traspaso posible, el juego
	// no se toca.
	for _, mio := range j.actual().Pacientes() {
		if !mio.Dormido() {
			continue
		}
		color := colorDeAtaque(mio.ataques[0])
	buscar:
		for _, rival := range j.rivales() {
			for _, suyo := range rival.Pacientes() {
				if suyo.Libre() && !ocupados[suyo] && color.combinaCon(suyo.Color()) {
					plan = append(plan, traspaso{mio, suyo})
					ocupados[suyo] = true
					break buscar
				}
			}
		}
	}
	if len(plan) == 0 {
		return ErrNadaQueChismear
	}
	for _, t := range plan {
		t.hacia.ataques, t.desde.ataques = t.desde.ataques, nil
	}
	j.anunciar("Chisme de la vecina: la señora del frente ya le contó a todo el barrio que a tu marido lo vieron dormido en casa ajena. %d ataque(s) cambiaron de dueño.", len(plan))
	j.descartar(c)
	return nil
}

// colorDeAtaque devuelve el color de una carta de ataque; los del Coste
// (nil) valen como multicolor.
func colorDeAtaque(c Carta) Color {
	if c == nil {
		return Multicolor
	}
	return c.Color()
}

// CartaPico ("Picó a todo volumen") hace que todos los rivales boten su mano
// y roben una nueva.
type CartaPico struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaPico) Tipo() TipoCarta { return TipoTratamiento }

// Requiere implementa Carta: no necesita objetivo.
func (c *CartaPico) Requiere() Requisitos { return Requisitos{} }

// Jugar descarta las manos de los rivales.
func (c *CartaPico) Jugar(j *Juego, _ Objetivo) error {
	j.descartar(c)
	for _, rival := range j.rivales() {
		j.descartar(rival.mano...)
		rival.mano = nil
		j.reponer(rival)
	}
	j.anunciar("%s subió el picó a todo volumen: nadie se oye, nadie piensa, todos los rivales botan su mano.", j.actual().nombre)
	return nil
}

// CartaCachos ("Te montaron los cachos") intercambia tu tablero con el de un
// rival, salvo los pacientes estables, que no se mueven.
type CartaCachos struct{ cartaBase }

// Tipo implementa Carta.
func (c *CartaCachos) Tipo() TipoCarta { return TipoTratamiento }

// Requiere implementa Carta: un rival.
func (c *CartaCachos) Requiere() Requisitos { return Requisitos{Rival: true} }

// Jugar intercambia los pacientes no estables de los dos tableros.
func (c *CartaCachos) Jugar(j *Juego, obj Objetivo) error {
	rival, err := j.rival(obj.Rival)
	if err != nil {
		return err
	}
	yo := j.actual()
	misMovibles, mioQueda := separarEstables(yo)
	susMovibles, suyoQueda := separarEstables(rival)
	if len(misMovibles) == 0 && len(susMovibles) == 0 {
		return ErrNadaQueCambiar
	}
	for _, p := range susMovibles {
		if _, ya := mioQueda[p.Color()]; ya {
			return ErrColorRepetido
		}
	}
	for _, p := range misMovibles {
		if _, ya := suyoQueda[p.Color()]; ya {
			return ErrColorRepetido
		}
	}
	for _, p := range susMovibles {
		mioQueda[p.Color()] = p
	}
	for _, p := range misMovibles {
		suyoQueda[p.Color()] = p
	}
	yo.tablero, rival.tablero = mioQueda, suyoQueda
	j.anunciar("¡%s, te montaron los cachos! Mientras dormías, %s se quedó con todo tu tablero. Pregúntale al Coste cómo se hace.",
		rival.nombre, yo.nombre)
	j.descartar(c)
	return nil
}

// separarEstables devuelve los pacientes que se pueden mover y un tablero
// nuevo solo con los estables, que se quedan.
func separarEstables(jug *Jugador) ([]*PacienteEnJuego, map[Color]*PacienteEnJuego) {
	var movibles []*PacienteEnJuego
	quedan := make(map[Color]*PacienteEnJuego)
	for _, p := range jug.Pacientes() {
		if p.Estable() {
			quedan[p.Color()] = p
		} else {
			movibles = append(movibles, p)
		}
	}
	return movibles, quedan
}
