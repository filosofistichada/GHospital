package hospital

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// escenarioPrueba agrupa un hospital armado y sus piezas para los tests.
type escenarioPrueba struct {
	h         *Hospital
	karen     *Doctor
	rafa      *Doctor
	kevin     *Camillero
	pacientes []*Paciente
}

// armarHospital crea un hospital con 2 doctores, 1 camillero (en ese orden
// de turno), las habitaciones pedidas de capacidad 1 y 4 pacientes, de los
// cuales el primero y el segundo son Severos.
func armarHospital(t *testing.T, habitaciones int) escenarioPrueba {
	t.Helper()
	e := escenarioPrueba{
		h:     NuevoHospital("Hospital de prueba"),
		karen: NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología"),
		rafa:  NuevoDoctor("D-002", "Dr. Rafa Escalona", 51, "Medicina interna"),
		kevin: NuevoCamillero("C-001", "Kevin Charris", 24),
		pacientes: []*Paciente{
			NuevoPaciente("P-001", "Wilfrido Berrío", 58, Severo),
			NuevoPaciente("P-002", "Yeimy Padilla", 34, Severo),
			NuevoPaciente("P-003", "Jhon Jairo Mercado", 41, Moderado),
			NuevoPaciente("P-004", "Dayana Barraza", 27, Leve),
		},
	}
	debe(t, e.h.ContratarDoctor(e.karen))
	debe(t, e.h.ContratarDoctor(e.rafa))
	debe(t, e.h.ContratarPersonal(e.kevin))
	for i := 0; i < habitaciones; i++ {
		hab, err := NuevaHabitacion(101+i, 1)
		debe(t, err)
		debe(t, e.h.AgregarHabitacion(hab))
	}
	for _, p := range e.pacientes {
		debe(t, e.h.AdmitirPaciente(p))
	}
	return e
}

// debe detiene el test si err no es nil.
func debe(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestAsignarHabitacionConCamaLibre(t *testing.T) {
	e := armarHospital(t, 1)
	p := e.pacientes[0]

	debe(t, p.SufrirAtaqueDeSueno("en la mecedora del pórtico"))
	hab, err := e.h.AsignarHabitacion(p)

	debe(t, err)
	if hab.Numero() != 101 {
		t.Errorf("habitación = %d, se esperaba 101", hab.Numero())
	}
	if p.Estado() != DormidoEnCama {
		t.Errorf("estado del paciente = %v, se esperaba %v", p.Estado(), DormidoEnCama)
	}
	if hab.Estado() != Ocupada {
		t.Errorf("estado de la habitación = %v, se esperaba %v", hab.Estado(), Ocupada)
	}
}

func TestAsignarHabitacionSinCama(t *testing.T) {
	e := armarHospital(t, 1)
	primero, segundo := e.pacientes[0], e.pacientes[1]

	debe(t, e.h.RegistrarEpisodio(primero, "frente al picó en la verbena"))
	err := e.h.RegistrarEpisodio(segundo, "en el Transmetro")

	if !errors.Is(err, ErrSinHabitacion) {
		t.Fatalf("error = %v, se esperaba ErrSinHabitacion", err)
	}
	if segundo.Estado() != DormidoEnPasillo {
		t.Errorf("estado = %v, el paciente debía seguir en el pasillo", segundo.Estado())
	}
	historial := e.h.Historial()
	if len(historial) != 2 {
		t.Fatalf("historial tiene %d episodios, se esperaban 2", len(historial))
	}
	if historial[1].Habitacion() != nil {
		t.Errorf("el episodio sin cama no debía tener habitación")
	}
}

func TestDespertarLiberaYReasignaActualizandoEpisodio(t *testing.T) {
	e := armarHospital(t, 1)
	primero, segundo := e.pacientes[0], e.pacientes[1]
	debe(t, e.h.RegistrarEpisodio(primero, "en la terraza a la hora de la siesta"))
	if err := e.h.RegistrarEpisodio(segundo, "en el Transmetro"); !errors.Is(err, ErrSinHabitacion) {
		t.Fatalf("error = %v, se esperaba ErrSinHabitacion", err)
	}

	debe(t, e.h.DespertarPaciente(primero))
	hab, err := e.h.AsignarHabitacion(segundo)

	debe(t, err)
	if hab.Numero() != 101 || segundo.Estado() != DormidoEnCama {
		t.Errorf("el segundo debía quedar en la 101 dormido en cama; quedó %v en %d", segundo.Estado(), hab.Numero())
	}
	if primero.Estado() != Despierto {
		t.Errorf("el primero debía quedar despierto, está %v", primero.Estado())
	}
	// Al segundo lo atendió Rafa (turno rotativo). Como Rafa y el hospital
	// comparten el mismo registro, Rafa ve la habitación asignada después.
	episodios := e.rafa.MisEpisodios()
	if len(episodios) != 1 {
		t.Fatalf("Rafa tiene %d episodios, se esperaba 1", len(episodios))
	}
	if episodios[0].Habitacion() == nil || episodios[0].Habitacion().Numero() != 101 {
		t.Errorf("el episodio de Rafa no se actualizó con la habitación 101")
	}
}

func TestDespertarPacienteDespierto(t *testing.T) {
	e := armarHospital(t, 1)

	err := e.h.DespertarPaciente(e.pacientes[0])

	if !errors.Is(err, ErrPacienteDespierto) {
		t.Errorf("error = %v, se esperaba ErrPacienteDespierto", err)
	}
}

func TestPacientesEnPasillo(t *testing.T) {
	e := armarHospital(t, 1)
	debe(t, e.h.RegistrarEpisodio(e.pacientes[0], "en la tienda de la esquina"))
	_ = e.h.RegistrarEpisodio(e.pacientes[2], "en la sala de espera de radiología")
	_ = e.h.RegistrarEpisodio(e.pacientes[3], "en el Transmetro")

	enPasillo := e.h.PacientesEnPasillo()

	if len(enPasillo) != 2 || enPasillo[0] != e.pacientes[2] || enPasillo[1] != e.pacientes[3] {
		t.Errorf("pasillo = %v, se esperaban P-003 y P-004", nombres(enPasillo))
	}
}

func TestReporteSeveros(t *testing.T) {
	e := armarHospital(t, 3)
	wilfrido, yeimy, jhon := e.pacientes[0], e.pacientes[1], e.pacientes[2]
	// Wilfrido se duerme dos veces, Jhon (Moderado) una, Yeimy ninguna.
	debe(t, e.h.RegistrarEpisodio(wilfrido, "en la mecedora"))
	debe(t, e.h.DespertarPaciente(wilfrido))
	debe(t, e.h.RegistrarEpisodio(wilfrido, "frente al picó"))
	debe(t, e.h.RegistrarEpisodio(jhon, "en la tienda"))

	casos := []struct {
		nombre string
		ahora  time.Time
		want   map[*Paciente]int
	}{
		{"hoy", time.Now(), map[*Paciente]int{wilfrido: 2, yeimy: 0}},
		{"mañana", time.Now().AddDate(0, 0, 1), map[*Paciente]int{wilfrido: 0, yeimy: 0}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e.h.ahora = func() time.Time { return c.ahora }

			got := e.h.ReporteSeveros()

			if len(got) != len(c.want) {
				t.Fatalf("el reporte tiene %d pacientes, se esperaban %d", len(got), len(c.want))
			}
			for p, n := range c.want {
				if got[p] != n {
					t.Errorf("%s: %d episodios, se esperaban %d", p.Nombre(), got[p], n)
				}
			}
		})
	}
}

func TestCamilleroNoDiagnostica(t *testing.T) {
	h := NuevoHospital("Solo camillero")
	kevin := NuevoCamillero("C-001", "Kevin Charris", 24)
	hab, err := NuevaHabitacion(101, 1)
	debe(t, err)
	p := NuevoPaciente("P-001", "Edinson Cantillo", 30, Severo)
	debe(t, h.ContratarPersonal(kevin))
	debe(t, h.AgregarHabitacion(hab))
	debe(t, h.AdmitirPaciente(p))

	debe(t, h.RegistrarEpisodio(p, "en la fila de la cafetería"))

	if p.DoctorAsignado() != nil {
		t.Errorf("el camillero no debía asignar doctor")
	}
	if p.Estado() != DormidoEnCama {
		t.Errorf("el camillero sí debía llevarlo a cama; estado = %v", p.Estado())
	}
}

func TestDespachoRotativoPolimorfico(t *testing.T) {
	e := armarHospital(t, 3)
	for _, p := range e.pacientes {
		_ = e.h.RegistrarEpisodio(p, "en el Transmetro")
	}

	// Personal: [Karen, Rafa, Kevin] -> el cuarto episodio vuelve a Karen.
	want := []string{"D-001", "D-002", "C-001", "D-001"}
	historial := e.h.Historial()
	if len(historial) != len(want) {
		t.Fatalf("historial tiene %d episodios, se esperaban %d", len(historial), len(want))
	}
	for i, r := range historial {
		if r.Atendedor().ID() != want[i] {
			t.Errorf("episodio %d atendido por %s, se esperaba %s", i+1, r.Atendedor().ID(), want[i])
		}
	}
	if n := len(e.karen.MisEpisodios()); n != 2 {
		t.Errorf("Karen tiene %d episodios, se esperaban 2", n)
	}
}

func TestDoctorSinCupoNoSeDespacha(t *testing.T) {
	h := NuevoHospital("Un solo doctor")
	karen := NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología")
	debe(t, h.ContratarDoctor(karen))
	for i := 0; i < cupoPacientesPorDoctor+1; i++ {
		p := NuevoPaciente(fmt.Sprintf("P-%03d", i), "Paciente", 30, Leve)
		debe(t, h.AdmitirPaciente(p))
		err := h.RegistrarEpisodio(p, "en el Transmetro")
		if i < cupoPacientesPorDoctor {
			if !errors.Is(err, ErrSinHabitacion) {
				t.Fatalf("paciente %d: error = %v, se esperaba ErrSinHabitacion", i, err)
			}
			continue
		}
		if !errors.Is(err, ErrSinPersonal) {
			t.Errorf("con el cupo lleno, error = %v, se esperaba ErrSinPersonal", err)
		}
	}
}

func TestErroresDeAlta(t *testing.T) {
	h := NuevoHospital("Altas")
	p := NuevoPaciente("P-001", "Yorladis De la Hoz", 29, Leve)
	d := NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología")
	hab, err := NuevaHabitacion(101, 1)
	debe(t, err)
	debe(t, h.AdmitirPaciente(p))
	debe(t, h.ContratarDoctor(d))
	debe(t, h.AgregarHabitacion(hab))
	_, errCapacidad := NuevaHabitacion(102, 0)
	var doctorNil *Doctor

	casos := []struct {
		nombre string
		err    error
		want   error
	}{
		{"paciente duplicado", h.AdmitirPaciente(NuevoPaciente("P-001", "Otro", 30, Leve)), ErrPacienteDuplicado},
		{"paciente nil", h.AdmitirPaciente(nil), ErrPacienteNulo},
		{"doctor duplicado", h.ContratarDoctor(d), ErrPersonalDuplicado},
		{"doctor nil", h.ContratarDoctor(doctorNil), ErrPersonalNulo},
		{"habitación duplicada", h.AgregarHabitacion(hab), ErrHabitacionDuplicada},
		{"capacidad inválida", errCapacidad, ErrCapacidadInvalida},
		{"paciente no admitido", h.RegistrarEpisodio(NuevoPaciente("P-999", "El Coste", 45, Leve), "donde la moza"), ErrPacienteNoAdmitido},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if !errors.Is(c.err, c.want) {
				t.Errorf("error = %v, se esperaba %v", c.err, c.want)
			}
		})
	}
}

func TestStringDeEstados(t *testing.T) {
	casos := []struct {
		got, want string
	}{
		{Despierto.String(), "despierto"},
		{DormidoEnPasillo.String(), "dormido en el pasillo"},
		{DormidoEnCama.String(), "dormido en cama"},
		{EstadoPaciente(99).String(), "desconocido"},
		{Severo.String(), "severo"},
		{NivelNarcolepsia(-1).String(), "desconocido"},
		{Ocupada.String(), "ocupada"},
	}
	for _, c := range casos {
		if c.got != c.want {
			t.Errorf("String() = %q, se esperaba %q", c.got, c.want)
		}
	}
}

func TestAsignacionConcurrenteNuncaSuperaCapacidad(t *testing.T) {
	const numPacientes, numHabitaciones, capacidad = 50, 3, 2
	h := NuevoHospital("Hospital concurrido")
	for i := 0; i < numHabitaciones; i++ {
		hab, err := NuevaHabitacion(101+i, capacidad)
		debe(t, err)
		debe(t, h.AgregarHabitacion(hab))
	}
	pacientes := make([]*Paciente, numPacientes)
	for i := range pacientes {
		pacientes[i] = NuevoPaciente(fmt.Sprintf("P-%03d", i), "Paciente", 30, Leve)
		debe(t, h.AdmitirPaciente(pacientes[i]))
		debe(t, pacientes[i].SufrirAtaqueDeSueno("en el Transmetro"))
	}

	// Todas las goroutines esperan la señal de salida para pedir cama a la
	// vez y maximizar la competencia por las habitaciones.
	var (
		wg        sync.WaitGroup
		salida    = make(chan struct{})
		asignados atomic.Int32
		sinCama   atomic.Int32
	)
	for _, p := range pacientes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-salida
			_, err := h.AsignarHabitacion(p)
			switch {
			case err == nil:
				asignados.Add(1)
			case errors.Is(err, ErrSinHabitacion):
				sinCama.Add(1)
			default:
				t.Errorf("error inesperado: %v", err)
			}
		}()
	}
	close(salida)
	wg.Wait()

	if got, want := int(asignados.Load()), numHabitaciones*capacidad; got != want {
		t.Errorf("asignados = %d, se esperaban %d", got, want)
	}
	if got, want := int(sinCama.Load()), numPacientes-numHabitaciones*capacidad; got != want {
		t.Errorf("sin cama = %d, se esperaban %d", got, want)
	}
	for _, hab := range h.Habitaciones() {
		if n := len(hab.Ocupantes()); n > hab.Capacidad() {
			t.Errorf("habitación %d con %d ocupantes y capacidad %d", hab.Numero(), n, hab.Capacidad())
		}
	}
}

// nombres facilita leer los mensajes de error de los tests.
func nombres(ps []*Paciente) []string {
	var ns []string
	for _, p := range ps {
		ns = append(ns, p.ID())
	}
	return ns
}
