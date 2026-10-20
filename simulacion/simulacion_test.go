package simulacion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"costenos-narcolepsia/hospital"
)

// armarHospital crea un hospital con más pacientes que camas, para que haya
// competencia por las habitaciones.
func armarHospital(t *testing.T, numPacientes, numHabitaciones int) *hospital.Hospital {
	t.Helper()
	h := hospital.NuevoHospital("Hospital simulado")
	if err := h.ContratarDoctor(hospital.NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología")); err != nil {
		t.Fatal(err)
	}
	if err := h.ContratarPersonal(hospital.NuevoCamillero("C-001", "Kevin Charris", 24)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < numHabitaciones; i++ {
		hab, err := hospital.NuevaHabitacion(101+i, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.AgregarHabitacion(hab); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < numPacientes; i++ {
		p := hospital.NuevoPaciente(fmt.Sprintf("P-%03d", i), "Paciente", 30, hospital.Severo)
		if err := h.AdmitirPaciente(p); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// configRapida acorta todos los tiempos para que el test dure poco.
func configRapida(duracion time.Duration) Config {
	return Config{
		Duracion:     duracion,
		DespiertoMax: 20 * time.Millisecond,
		DormidoMax:   30 * time.Millisecond,
		RondaPasillo: 5 * time.Millisecond,
		Ubicaciones:  []string{"en el Transmetro", "en la terraza"},
	}
}

func TestSimulacionRespetaCapacidadYCuentaBien(t *testing.T) {
	h := armarHospital(t, 10, 2)
	var eventos []Evento

	res := Ejecutar(context.Background(), h, configRapida(300*time.Millisecond), func(ev Evento) {
		eventos = append(eventos, ev) // seguro: el callback corre en una sola goroutine
	})

	if res.Ataques == 0 {
		t.Fatal("no hubo ningún ataque de sueño")
	}
	if res.Ataques != res.ConCama+res.EnPasillo {
		t.Errorf("ataques = %d, pero con cama + en pasillo = %d", res.Ataques, res.ConCama+res.EnPasillo)
	}
	if res.Fallos != 0 {
		t.Errorf("hubo %d fallos inesperados", res.Fallos)
	}
	total := res.Ataques + res.LlevadosPorRonda + res.Despertares + res.Fallos
	if len(eventos) != total {
		t.Errorf("llegaron %d eventos al callback, el resumen cuenta %d", len(eventos), total)
	}
	if got := len(h.Historial()); got != res.Ataques {
		t.Errorf("el historial tiene %d episodios, el resumen cuenta %d ataques", got, res.Ataques)
	}

	// Ya terminaron las goroutines: leer el estado es seguro.
	enCama := 0
	for _, p := range h.Pacientes() {
		if p.Estado() == hospital.DormidoEnCama {
			enCama++
		}
	}
	ocupantes := 0
	for _, hab := range h.Habitaciones() {
		n := len(hab.Ocupantes())
		if n > hab.Capacidad() {
			t.Errorf("habitación %d con %d ocupantes y capacidad %d", hab.Numero(), n, hab.Capacidad())
		}
		ocupantes += n
	}
	if enCama != ocupantes {
		t.Errorf("%d pacientes dicen estar en cama, pero las habitaciones tienen %d ocupantes", enCama, ocupantes)
	}
	// Gracias al cierre de turno, nadie queda en el pasillo con camas libres.
	if len(h.PacientesEnPasillo()) > 0 && ocupantes < 2 {
		t.Errorf("quedaron pacientes en el pasillo con %d de 2 camas ocupadas", ocupantes)
	}
}

func TestSimulacionSeDetieneAlCancelar(t *testing.T) {
	h := armarHospital(t, 5, 1)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	inicio := time.Now()
	Ejecutar(ctx, h, configRapida(10*time.Second), nil)

	if d := time.Since(inicio); d > time.Second {
		t.Errorf("con el contexto cancelado tardó %v; debía terminar de inmediato", d)
	}
}
