package simulacion

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"costenos-narcolepsia/hospital"
)

// TipoEvento clasifica lo que pasó durante la simulación.
type TipoEvento int

// Tipos de evento.
const (
	SeDurmioEnCama    TipoEvento = iota // se durmió y consiguió cama de una vez
	SeDurmioEnPasillo                   // se durmió y no había cama
	LlevadoACama                        // la ronda del pasillo le consiguió cama después
	Desperto                            // se despertó y liberó su cama (si tenía)
	Fallo                               // una operación devolvió un error inesperado
)

// String devuelve el tipo de evento en texto legible.
func (t TipoEvento) String() string {
	switch t {
	case SeDurmioEnCama:
		return "se durmió en cama"
	case SeDurmioEnPasillo:
		return "se durmió en el pasillo"
	case LlevadoACama:
		return "llevado a cama"
	case Desperto:
		return "se despertó"
	case Fallo:
		return "fallo"
	default:
		return "desconocido"
	}
}

// Evento describe algo que pasó durante la simulación.
type Evento struct {
	Momento    time.Duration // tiempo desde el inicio de la simulación
	Tipo       TipoEvento
	Paciente   *hospital.Paciente
	Ubicacion  string // dónde se durmió (SeDurmioEnCama y SeDurmioEnPasillo)
	Habitacion int    // número de habitación (LlevadoACama)
	Err        error  // error recibido (SeDurmioEnPasillo y Fallo)
}

// Config ajusta la simulación. Los campos en cero toman un valor por defecto.
type Config struct {
	Duracion     time.Duration // cuánto dura la simulación (5 s por defecto)
	DespiertoMax time.Duration // tiempo máximo despierto entre ataques
	DormidoMax   time.Duration // tiempo máximo dormido antes de despertar
	RondaPasillo time.Duration // cada cuánto se revisa el pasillo
	Ubicaciones  []string      // lugares donde les puede dar el ataque
}

// Resumen cuenta los eventos de una simulación.
type Resumen struct {
	Ataques          int // episodios registrados (con o sin cama)
	ConCama          int // ataques que consiguieron cama de inmediato
	EnPasillo        int // ataques que se quedaron sin cama
	LlevadosPorRonda int // pacientes del pasillo que recibieron cama después
	Despertares      int
	Fallos           int
}

// contar suma un evento al resumen.
func (r *Resumen) contar(ev Evento) {
	switch ev.Tipo {
	case SeDurmioEnCama:
		r.Ataques++
		r.ConCama++
	case SeDurmioEnPasillo:
		r.Ataques++
		r.EnPasillo++
	case LlevadoACama:
		r.LlevadosPorRonda++
	case Desperto:
		r.Despertares++
	case Fallo:
		r.Fallos++
	}
}

// Ejecutar corre la simulación sobre el hospital: lanza una goroutine por
// paciente (que se duerme y se despierta a intervalos aleatorios) y otra que
// hace la ronda del pasillo. Termina cuando pasa cfg.Duracion o se cancela
// ctx, espera a que todas las goroutines acaben y devuelve el resumen.
//
// alEvento (puede ser nil) se llama por cada evento, siempre desde la
// goroutine que llamó a Ejecutar, así que no necesita sincronización.
// Todos los cambios de estado pasan por métodos del Hospital, que es el
// único que tiene mutex.
func Ejecutar(ctx context.Context, h *hospital.Hospital, cfg Config, alEvento func(Evento)) Resumen {
	cfg = cfg.conValoresPorDefecto()
	ctx, cancelar := context.WithTimeout(ctx, cfg.Duracion)
	defer cancelar()

	inicio := time.Now()
	eventos := make(chan Evento, 64)
	emitir := func(ev Evento) {
		ev.Momento = time.Since(inicio)
		eventos <- ev
	}

	var wg sync.WaitGroup
	for _, p := range h.Pacientes() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vivir(ctx, h, p, cfg, emitir)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		rondarPasillo(ctx, h, cfg, emitir)
	}()
	go func() {
		wg.Wait()
		// Cierre de turno: ya sin goroutines, una última ronda para no dejar
		// a nadie en el pasillo si quedaron camas libres. Después se cierra
		// el canal para que el range de abajo termine.
		revisarPasillo(h, emitir)
		close(eventos)
	}()

	var resumen Resumen
	for ev := range eventos {
		resumen.contar(ev)
		if alEvento != nil {
			alEvento(ev)
		}
	}
	return resumen
}

// vivir es el ciclo de un paciente: espera despierto, le da el ataque, duerme
// un rato y se despierta, hasta que se acaba el tiempo.
func vivir(ctx context.Context, h *hospital.Hospital, p *hospital.Paciente, cfg Config, emitir func(Evento)) {
	for {
		if !esperar(ctx, alAzar(cfg.DespiertoMax)) {
			return
		}
		ubicacion := cfg.Ubicaciones[rand.IntN(len(cfg.Ubicaciones))]
		err := h.RegistrarEpisodio(p, ubicacion)
		switch {
		case err == nil:
			emitir(Evento{Tipo: SeDurmioEnCama, Paciente: p, Ubicacion: ubicacion})
		case errors.Is(err, hospital.ErrSinHabitacion):
			emitir(Evento{Tipo: SeDurmioEnPasillo, Paciente: p, Ubicacion: ubicacion, Err: err})
		default:
			emitir(Evento{Tipo: Fallo, Paciente: p, Err: err})
			continue
		}

		if !esperar(ctx, alAzar(cfg.DormidoMax)) {
			return
		}
		if err := h.DespertarPaciente(p); err != nil {
			emitir(Evento{Tipo: Fallo, Paciente: p, Err: err})
			continue
		}
		emitir(Evento{Tipo: Desperto, Paciente: p})
	}
}

// rondarPasillo revisa el pasillo cada cierto tiempo hasta que se acabe la
// simulación.
func rondarPasillo(ctx context.Context, h *hospital.Hospital, cfg Config, emitir func(Evento)) {
	for esperar(ctx, cfg.RondaPasillo) {
		revisarPasillo(h, emitir)
	}
}

// revisarPasillo hace lo que motiva la consulta PacientesEnPasillo: busca a
// los dormidos en el pasillo e intenta darles cama.
func revisarPasillo(h *hospital.Hospital, emitir func(Evento)) {
	for _, p := range h.PacientesEnPasillo() {
		// Entre la consulta y la asignación el paciente pudo despertarse u
		// otra goroutine pudo ocupar la cama: AsignarHabitacion lo vuelve a
		// validar con el lock tomado y devuelve error.
		hab, err := h.AsignarHabitacion(p)
		if err != nil {
			continue
		}
		emitir(Evento{Tipo: LlevadoACama, Paciente: p, Habitacion: hab.Numero()})
	}
}

// esperar duerme d o hasta que se cancele ctx. Devuelve false si se canceló.
func esperar(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// alAzar devuelve una duración aleatoria entre max/10 y max.
// math/rand/v2 es seguro para uso concurrente desde varias goroutines.
func alAzar(max time.Duration) time.Duration {
	return max/10 + rand.N(max-max/10)
}

// conValoresPorDefecto completa los campos en cero.
func (c Config) conValoresPorDefecto() Config {
	if c.Duracion <= 0 {
		c.Duracion = 5 * time.Second
	}
	if c.DespiertoMax <= 0 {
		c.DespiertoMax = 1200 * time.Millisecond
	}
	if c.DormidoMax <= 0 {
		c.DormidoMax = 1500 * time.Millisecond
	}
	if c.RondaPasillo <= 0 {
		c.RondaPasillo = 250 * time.Millisecond
	}
	if len(c.Ubicaciones) == 0 {
		c.Ubicaciones = []string{"en el pasillo"}
	}
	return c
}
