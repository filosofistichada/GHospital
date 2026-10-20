package virus

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"
)

// ---------- Ayudas ----------

// juegoVacio crea una partida con semilla fija y vacía manos y tableros para
// armar cada caso a mano. El jugador 0 (Sofía) tiene el turno.
func juegoVacio(t *testing.T, numBots int) *Juego {
	t.Helper()
	j, err := NuevoJuego("Sofía", numBots, rand.New(rand.NewPCG(1, 2)))
	if err != nil {
		t.Fatal(err)
	}
	for _, jug := range j.jugadores {
		jug.mano = nil
		jug.tablero = make(map[Color]*PacienteEnJuego)
	}
	return j
}

func cartaPaciente(c Color) *CartaPaciente { return &CartaPaciente{cartaBase{"paciente", "", c}} }
func cartaAtaque(c Color) *CartaAtaque     { return &CartaAtaque{cartaBase{"ataque", "", c}} }
func cartaCama(c Color) *CartaCama         { return &CartaCama{cartaBase{"cama", "", c}} }

// poner coloca en el tablero un paciente con los ataques y camas pedidos.
func poner(jug *Jugador, c Color, ataques, camas int) *PacienteEnJuego {
	p := &PacienteEnJuego{carta: cartaPaciente(c)}
	for i := 0; i < ataques; i++ {
		p.ataques = append(p.ataques, cartaAtaque(c))
	}
	for i := 0; i < camas; i++ {
		p.camas = append(p.camas, cartaCama(c))
	}
	jug.tablero[c] = p
	return p
}

// jugar le da la carta al jugador de turno y la juega.
func jugar(j *Juego, carta Carta, obj Objetivo) error {
	j.actual().mano = []Carta{carta}
	return j.JugarCarta(0, obj)
}

// contarCartas cuenta todas las cartas del juego y falla si alguna aparece
// dos veces.
func contarCartas(t *testing.T, j *Juego) int {
	t.Helper()
	vistas := make(map[Carta]bool)
	agregar := func(cs ...Carta) {
		for _, c := range cs {
			if vistas[c] {
				t.Fatalf("la carta %q aparece dos veces", c.Nombre())
			}
			vistas[c] = true
		}
	}
	agregar(j.mazo...)
	agregar(j.descarte...)
	for _, jug := range j.jugadores {
		agregar(jug.mano...)
		for _, p := range jug.tablero {
			agregar(p.todasLasCartas()...)
		}
	}
	return len(vistas)
}

// ---------- Mazo y reparto ----------

func TestMazoCompleto(t *testing.T) {
	porTipo := make(map[TipoCarta]int)
	for _, c := range nuevoMazo() {
		porTipo[c.Tipo()]++
	}
	want := map[TipoCarta]int{TipoPaciente: 21, TipoAtaque: 18, TipoCama: 20, TipoTratamiento: 9} // ataques: 17 + el Coste
	total := 0
	for tipo, n := range want {
		if porTipo[tipo] != n {
			t.Errorf("%v: %d cartas, se esperaban %d", tipo, porTipo[tipo], n)
		}
		total += porTipo[tipo]
	}
	if total != TamanoMazo {
		t.Errorf("el mazo tiene %d cartas, se esperaban %d", total, TamanoMazo)
	}
}

func TestNuevoJuegoReparte(t *testing.T) {
	if _, err := NuevoJuego("Sofía", 4, rand.New(rand.NewPCG(1, 2))); !errors.Is(err, ErrNumeroBots) {
		t.Errorf("con 4 bots: error = %v, se esperaba ErrNumeroBots", err)
	}
	j, err := NuevoJuego("Sofía", 3, rand.New(rand.NewPCG(1, 2)))
	if err != nil {
		t.Fatal(err)
	}
	for _, jug := range j.Jugadores() {
		if len(jug.Mano()) != cartasEnMano {
			t.Errorf("%s tiene %d cartas, se esperaban %d", jug.Nombre(), len(jug.Mano()), cartasEnMano)
		}
	}
	if n := contarCartas(t, j); n != TamanoMazo {
		t.Errorf("hay %d cartas en juego, se esperaban %d", n, TamanoMazo)
	}
}

// ---------- Ataques y camas ----------

func TestAtaque(t *testing.T) {
	casos := []struct {
		nombre             string
		ataque             Color
		paciente           Color
		ataques, camas     int
		wantErr            error
		wantEnTablero      bool
		wantDormido        bool
		wantCamasDespues   int
		wantMensajeContien string
	}{
		{"libre se duerme", Rojo, Rojo, 0, 0, nil, true, true, 0, ""},
		{"dormido se va pa' Barranquilla", Rojo, Rojo, 1, 0, nil, false, false, 0, "Barranquilla"},
		{"la cama anula el ataque", Azul, Azul, 0, 1, nil, true, false, 0, ""},
		{"estable es inmune", Verde, Verde, 0, 2, ErrPacienteEstable, true, false, 2, ""},
		{"color que no combina", Rojo, Azul, 0, 0, ErrColorIncompatible, true, false, 0, ""},
		{"comida comodín sirve en cualquiera", Multicolor, Amarillo, 0, 0, nil, true, true, 0, ""},
		{"cualquier comida sirve en el paciente comodín", Rojo, Multicolor, 0, 0, nil, true, true, 0, ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			j := juegoVacio(t, 1)
			rival := j.jugadores[1]
			poner(rival, c.paciente, c.ataques, c.camas)

			err := jugar(j, cartaAtaque(c.ataque), Objetivo{Rival: 1, PacienteRival: c.paciente})

			if !errors.Is(err, c.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, c.wantErr)
			}
			p, enTablero := rival.tablero[c.paciente]
			if enTablero != c.wantEnTablero {
				t.Fatalf("en tablero = %v, se esperaba %v", enTablero, c.wantEnTablero)
			}
			if enTablero && (p.Dormido() != c.wantDormido || p.Camas() != c.wantCamasDespues) {
				t.Errorf("dormido = %v y camas = %d; se esperaba %v y %d", p.Dormido(), p.Camas(), c.wantDormido, c.wantCamasDespues)
			}
			if c.wantMensajeContien != "" && !strings.Contains(strings.Join(j.TomarMensajes(), " "), c.wantMensajeContien) {
				t.Errorf("faltó el mensaje con %q", c.wantMensajeContien)
			}
		})
	}
}

func TestCamas(t *testing.T) {
	casos := []struct {
		nombre         string
		carta          Carta
		ataques, camas int
		wantErr        error
		wantDormido    bool
		wantCamas      int
	}{
		{"cama protege al libre", cartaCama(Rojo), 0, 0, nil, false, 1},
		{"dos camas: estable", cartaCama(Rojo), 0, 1, nil, false, 2},
		{"cama despierta al dormido", cartaCama(Rojo), 1, 0, nil, false, 0},
		{"estable no necesita más", cartaCama(Rojo), 0, 2, ErrPacienteEstable, false, 2},
		{"color que no combina", cartaCama(Azul), 0, 0, ErrColorIncompatible, false, 0},
		{"doctor despierta al dormido", &CartaDoctor{cartaBase{"doctor", "", Multicolor}}, 1, 0, nil, false, 0},
		{"camillero protege al despierto", &CartaCamillero{cartaBase{"camillero", "", Multicolor}}, 0, 0, nil, false, 1},
		{"camillero no diagnostica al dormido", &CartaCamillero{cartaBase{"camillero", "", Multicolor}}, 1, 0, ErrCamilleroNoDiagnost, true, 0},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			j := juegoVacio(t, 1)
			p := poner(j.jugadores[0], Rojo, c.ataques, c.camas)

			err := jugar(j, c.carta, Objetivo{PacientePropio: Rojo})

			if !errors.Is(err, c.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, c.wantErr)
			}
			if p.Dormido() != c.wantDormido || p.Camas() != c.wantCamas {
				t.Errorf("dormido = %v y camas = %d; se esperaba %v y %d", p.Dormido(), p.Camas(), c.wantDormido, c.wantCamas)
			}
		})
	}
}

func TestJugadaFallidaNoCambiaNada(t *testing.T) {
	j := juegoVacio(t, 1)
	poner(j.jugadores[0], Rojo, 0, 0)
	carta := cartaPaciente(Rojo)

	err := jugar(j, carta, Objetivo{})

	if !errors.Is(err, ErrColorRepetido) {
		t.Fatalf("error = %v, se esperaba ErrColorRepetido", err)
	}
	if j.Turno() != 0 || len(j.jugadores[0].mano) != 1 || j.jugadores[0].mano[0] != carta {
		t.Errorf("tras una jugada inválida cambió el turno o la mano")
	}
}

// ---------- El Coste ----------

func TestCosteRespetaEstables(t *testing.T) {
	j := juegoVacio(t, 1)
	rival := j.jugadores[1]
	estable := poner(rival, Rojo, 0, 2)
	libre := poner(rival, Azul, 0, 0)
	poner(rival, Verde, 1, 0) // dormido: se va del tablero
	conCama := poner(rival, Amarillo, 0, 1)

	debe(t, jugar(j, &CartaCoste{cartaBase{"El Coste", "", Multicolor}}, Objetivo{Rival: 1}))

	if !estable.Estable() || estable.Dormido() {
		t.Errorf("el estable debía quedar intacto")
	}
	if !libre.Dormido() {
		t.Errorf("el libre debía quedar dormido")
	}
	if _, sigue := rival.tablero[Verde]; sigue {
		t.Errorf("el que ya dormía debía irse pa' Barranquilla")
	}
	if conCama.Camas() != 0 || conCama.Dormido() {
		t.Errorf("la cama y el ataque del Coste debían anularse")
	}
	mensajes := strings.Join(j.TomarMensajes(), " ")
	if !strings.Contains(mensajes, "el Coste") {
		t.Errorf("faltó la frase del Coste en %q", mensajes)
	}
}

func TestCosteSinVictimas(t *testing.T) {
	j := juegoVacio(t, 1)
	poner(j.jugadores[1], Rojo, 0, 2)

	err := jugar(j, &CartaCoste{cartaBase{"El Coste", "", Multicolor}}, Objetivo{Rival: 1})

	if !errors.Is(err, ErrCosteSinVictimas) {
		t.Errorf("error = %v, se esperaba ErrCosteSinVictimas", err)
	}
}

// ---------- Tratamientos ----------

func TestLaMoza(t *testing.T) {
	moza := func() Carta { return &CartaLaMoza{cartaBase{"La moza", "", Multicolor}} }

	t.Run("roba un paciente", func(t *testing.T) {
		j := juegoVacio(t, 1)
		suyo := poner(j.jugadores[1], Rojo, 1, 0)
		debe(t, jugar(j, moza(), Objetivo{Rival: 1, PacienteRival: Rojo}))
		if j.jugadores[0].tablero[Rojo] != suyo || len(j.jugadores[1].tablero) != 0 {
			t.Errorf("el paciente debía pasar al tablero de Sofía con su ataque")
		}
	})
	t.Run("no roba estables", func(t *testing.T) {
		j := juegoVacio(t, 1)
		poner(j.jugadores[1], Rojo, 0, 2)
		if err := jugar(j, moza(), Objetivo{Rival: 1, PacienteRival: Rojo}); !errors.Is(err, ErrPacienteEstable) {
			t.Errorf("error = %v, se esperaba ErrPacienteEstable", err)
		}
	})
	t.Run("no deja dos del mismo color", func(t *testing.T) {
		j := juegoVacio(t, 1)
		poner(j.jugadores[0], Rojo, 0, 0)
		poner(j.jugadores[1], Rojo, 0, 0)
		if err := jugar(j, moza(), Objetivo{Rival: 1, PacienteRival: Rojo}); !errors.Is(err, ErrColorRepetido) {
			t.Errorf("error = %v, se esperaba ErrColorRepetido", err)
		}
	})
}

func TestCambioDeNovia(t *testing.T) {
	j := juegoVacio(t, 1)
	mio := poner(j.jugadores[0], Rojo, 1, 0)
	suyo := poner(j.jugadores[1], Azul, 0, 1)

	debe(t, jugar(j, &CartaCambioDeNovia{cartaBase{"Novia", "", Multicolor}}, Objetivo{Rival: 1, PacienteRival: Azul, PacientePropio: Rojo}))

	if j.jugadores[0].tablero[Azul] != suyo || j.jugadores[1].tablero[Rojo] != mio {
		t.Errorf("los pacientes debían intercambiarse con lo que tenían encima")
	}
	if len(j.jugadores[0].tablero) != 1 || len(j.jugadores[1].tablero) != 1 {
		t.Errorf("cada uno debía quedar con un solo paciente")
	}
}

func TestCachosNoMueveEstables(t *testing.T) {
	j := juegoVacio(t, 1)
	yo, rival := j.jugadores[0], j.jugadores[1]
	miEstable := poner(yo, Rojo, 0, 2)
	miMovible := poner(yo, Azul, 0, 0)
	suMovible := poner(rival, Verde, 1, 0)

	debe(t, jugar(j, &CartaCachos{cartaBase{"Cachos", "", Multicolor}}, Objetivo{Rival: 1}))

	if yo.tablero[Rojo] != miEstable || yo.tablero[Verde] != suMovible || len(yo.tablero) != 2 {
		t.Errorf("Sofía debía quedarse con su estable y recibir el verde")
	}
	if rival.tablero[Azul] != miMovible || len(rival.tablero) != 1 {
		t.Errorf("el rival debía recibir el azul")
	}
}

func TestCachosSinColoresRepetidos(t *testing.T) {
	j := juegoVacio(t, 1)
	poner(j.jugadores[0], Rojo, 0, 2) // estable, se queda
	poner(j.jugadores[1], Rojo, 0, 0) // vendría a dar al lado del estable rojo

	err := jugar(j, &CartaCachos{cartaBase{"Cachos", "", Multicolor}}, Objetivo{Rival: 1})

	if !errors.Is(err, ErrColorRepetido) {
		t.Errorf("error = %v, se esperaba ErrColorRepetido", err)
	}
}

func TestChismeDeLaVecina(t *testing.T) {
	j := juegoVacio(t, 2)
	mio := poner(j.jugadores[0], Rojo, 1, 0)
	conCama := poner(j.jugadores[1], Rojo, 0, 1) // no está libre
	libre := poner(j.jugadores[2], Rojo, 0, 0)

	debe(t, jugar(j, &CartaChisme{cartaBase{"Chisme", "", Multicolor}}, Objetivo{}))

	if mio.Dormido() || !libre.Dormido() || conCama.Dormido() {
		t.Errorf("el ataque debía pasar solo al paciente libre del mismo color")
	}
}

func TestChismeSinNadaQuePasar(t *testing.T) {
	j := juegoVacio(t, 1)
	poner(j.jugadores[1], Rojo, 0, 0)

	err := jugar(j, &CartaChisme{cartaBase{"Chisme", "", Multicolor}}, Objetivo{})

	if !errors.Is(err, ErrNadaQueChismear) {
		t.Errorf("error = %v, se esperaba ErrNadaQueChismear", err)
	}
}

func TestPicoATodoVolumen(t *testing.T) {
	j := juegoVacio(t, 2)
	viejas := []Carta{cartaCama(Rojo), cartaCama(Azul), cartaCama(Verde)}
	j.jugadores[1].mano = append([]Carta(nil), viejas...)

	debe(t, jugar(j, &CartaPico{cartaBase{"Picó", "", Multicolor}}, Objetivo{}))

	nueva := j.jugadores[1].mano
	if len(nueva) != cartasEnMano {
		t.Fatalf("el rival quedó con %d cartas, se esperaban %d", len(nueva), cartasEnMano)
	}
	for _, c := range nueva {
		for _, v := range viejas {
			if c == v {
				t.Errorf("el rival conservó una carta que debía botar")
			}
		}
	}
}

// ---------- Turnos y victoria ----------

func TestVictoria(t *testing.T) {
	t.Run("cuatro sanos ganan", func(t *testing.T) {
		j := juegoVacio(t, 1)
		yo := j.jugadores[0]
		poner(yo, Rojo, 0, 0)
		poner(yo, Azul, 0, 1)
		poner(yo, Verde, 0, 2) // estable cuenta como sano
		debe(t, jugar(j, cartaPaciente(Multicolor), Objetivo{}))
		if j.Ganador() != yo {
			t.Fatalf("Sofía debía ganar")
		}
		if err := jugar(j, cartaPaciente(Amarillo), Objetivo{}); !errors.Is(err, ErrJuegoTerminado) {
			t.Errorf("después de ganar: error = %v, se esperaba ErrJuegoTerminado", err)
		}
	})
	t.Run("dormido no cuenta", func(t *testing.T) {
		j := juegoVacio(t, 1)
		yo := j.jugadores[0]
		poner(yo, Rojo, 0, 0)
		poner(yo, Azul, 0, 0)
		poner(yo, Verde, 1, 0)
		debe(t, jugar(j, cartaPaciente(Amarillo), Objetivo{}))
		if j.Ganador() != nil {
			t.Errorf("con un paciente dormido no se gana")
		}
	})
}

func TestDescartar(t *testing.T) {
	j := juegoVacio(t, 1)
	j.jugadores[0].mano = []Carta{cartaCama(Rojo), cartaCama(Azul), cartaCama(Verde)}

	for _, malos := range [][]int{{}, {0, 0}, {5}, {0, 1, 2, 0}} {
		if err := j.Descartar(malos...); !errors.Is(err, ErrDescarteInvalido) {
			t.Errorf("Descartar(%v): error = %v, se esperaba ErrDescarteInvalido", malos, err)
		}
	}
	debe(t, j.Descartar(0, 2))

	if j.Turno() != 1 || len(j.jugadores[0].mano) != cartasEnMano {
		t.Errorf("tras descartar debía reponer la mano y pasar el turno")
	}
}

// ---------- Bots ----------

func TestPartidasEntreBotsConservanLasCartas(t *testing.T) {
	terminadas := 0
	for semilla := uint64(1); semilla <= 30; semilla++ {
		j, err := NuevoJuego("Sofía", 3, rand.New(rand.NewPCG(semilla, 7)))
		if err != nil {
			t.Fatal(err)
		}
		j.jugadores[0].bot = &perfilesBot[0] // Sofía también juega sola
		for turno := 0; turno < 2000 && j.Ganador() == nil; turno++ {
			if err := j.JugarTurnoBot(); err != nil {
				t.Fatalf("semilla %d, turno %d: %v", semilla, turno, err)
			}
			if n := contarCartas(t, j); n != TamanoMazo {
				t.Fatalf("semilla %d, turno %d: hay %d cartas, se esperaban %d", semilla, turno, n, TamanoMazo)
			}
		}
		if j.Ganador() != nil {
			terminadas++
		}
	}
	t.Logf("%d de 30 partidas terminaron con ganador", terminadas)
	if terminadas == 0 {
		t.Errorf("ninguna partida entre bots terminó con un ganador")
	}
}

func TestMismaSemillaMismaPartida(t *testing.T) {
	partida := func() []string {
		j, err := NuevoJuego("Sofía", 2, rand.New(rand.NewPCG(42, 42)))
		if err != nil {
			t.Fatal(err)
		}
		j.jugadores[0].bot = &perfilesBot[2]
		var registro []string
		for i := 0; i < 50 && j.Ganador() == nil; i++ {
			debe(t, j.JugarTurnoBot())
			registro = append(registro, j.TomarMensajes()...)
		}
		return registro
	}
	a, b := partida(), partida()
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Errorf("con la misma semilla las partidas debían ser idénticas")
	}
}

func debe(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}
