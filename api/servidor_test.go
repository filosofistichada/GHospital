package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"costenos-narcolepsia/hospital"
)

// armarHospitalPrueba replica el escenario: 2 doctores, 1 camillero,
// 3 habitaciones de capacidad 1 y 5 pacientes (P-001, P-002 y P-005 Severos).
func armarHospitalPrueba() (*hospital.Hospital, map[string]string) {
	h := hospital.NuevoHospital("Hospital de prueba")
	_ = h.ContratarDoctor(hospital.NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología"))
	_ = h.ContratarDoctor(hospital.NuevoDoctor("D-002", "Dr. Rafa Escalona", 51, "Medicina interna"))
	_ = h.ContratarPersonal(hospital.NuevoCamillero("C-001", "Kevin Charris", 24))
	for _, n := range []int{101, 102, 103} {
		hab, _ := hospital.NuevaHabitacion(n, 1)
		_ = h.AgregarHabitacion(hab)
	}
	niveles := []hospital.NivelNarcolepsia{hospital.Severo, hospital.Severo, hospital.Moderado, hospital.Leve, hospital.Severo}
	for i, nivel := range niveles {
		_ = h.AdmitirPaciente(hospital.NuevoPaciente(fmt.Sprintf("P-%03d", i+1), fmt.Sprintf("Paciente %d", i+1), 30, nivel))
	}
	return h, map[string]string{"P-001": "rojo", "P-002": "azul"}
}

func nuevoServidorPrueba() *Servidor {
	return NuevoServidor(Config{
		ArmarHospital: armarHospitalPrueba,
		Lugares:       []string{"en el Transmetro"},
		Archivos:      fstest.MapFS{"index.html": {Data: []byte("<h1>Hospital</h1>")}},
		NuevoAzar:     func() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) },
	})
}

// pedir hace una petición al servidor y decodifica la respuesta en v (si no
// es nil). Devuelve el código HTTP.
func pedir(t *testing.T, s http.Handler, metodo, ruta, cuerpo string, v any) int {
	t.Helper()
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if v != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
			t.Fatalf("%s %s: respuesta no es JSON válido: %v\n%s", metodo, ruta, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestEstadoInicial(t *testing.T) {
	s := nuevoServidorPrueba()
	var estado estadoDTO

	code := pedir(t, s, "GET", "/api/estado", "", &estado)

	if code != http.StatusOK {
		t.Fatalf("código = %d, se esperaba 200", code)
	}
	if len(estado.Pacientes) != 5 || len(estado.Habitaciones) != 3 || len(estado.Personal) != 3 {
		t.Errorf("se esperaban 5 pacientes, 3 habitaciones y 3 del personal; llegaron %d, %d y %d",
			len(estado.Pacientes), len(estado.Habitaciones), len(estado.Personal))
	}
	if estado.Pacientes[0].Equipo != "rojo" || estado.Pacientes[3].Equipo != "multicolor" {
		t.Errorf("equipos mal asignados: %q y %q", estado.Pacientes[0].Equipo, estado.Pacientes[3].Equipo)
	}
	if estado.Personal[0].Rol != "doctor" || estado.Personal[2].Rol != "apoyo" {
		t.Errorf("roles mal asignados: %q y %q", estado.Personal[0].Rol, estado.Personal[2].Rol)
	}
}

func TestAtaquesHastaQuedarseSinCama(t *testing.T) {
	s := nuevoServidorPrueba()
	for _, id := range []string{"P-001", "P-002", "P-003"} {
		var acc accionDTO
		if code := pedir(t, s, "POST", "/api/pacientes/"+id+"/ataque", `{"ubicacion":"en la terraza"}`, &acc); code != http.StatusOK || acc.Aviso != "" {
			t.Fatalf("%s: código %d, aviso %q; se esperaba 200 sin aviso", id, code, acc.Aviso)
		}
	}
	var acc accionDTO

	code := pedir(t, s, "POST", "/api/pacientes/P-005/ataque", "", &acc)

	if code != http.StatusOK {
		t.Fatalf("código = %d, se esperaba 200 (el episodio se registra igual)", code)
	}
	if !strings.Contains(acc.Aviso, "no hay cama") {
		t.Errorf("aviso = %q, se esperaba el de no hay cama", acc.Aviso)
	}
	if len(acc.Estado.Pasillo) != 1 || acc.Estado.Pasillo[0].ID != "P-005" {
		t.Errorf("en el pasillo debía quedar solo P-005: %+v", acc.Estado.Pasillo)
	}
	if acc.Estado.Pasillo[0].Ubicacion != "en el Transmetro" {
		t.Errorf("sin ubicación debía usar un lugar de la configuración, usó %q", acc.Estado.Pasillo[0].Ubicacion)
	}
}

func TestDespertarYReasignar(t *testing.T) {
	s := nuevoServidorPrueba()
	for _, id := range []string{"P-001", "P-002", "P-003", "P-005"} {
		pedir(t, s, "POST", "/api/pacientes/"+id+"/ataque", "", nil)
	}
	if code := pedir(t, s, "POST", "/api/pacientes/P-001/despertar", "", nil); code != http.StatusOK {
		t.Fatalf("despertar: código %d", code)
	}
	var acc accionDTO

	code := pedir(t, s, "POST", "/api/pacientes/P-005/cama", "", &acc)

	if code != http.StatusOK || len(acc.Estado.Pasillo) != 0 {
		t.Fatalf("código %d y pasillo %+v; se esperaba 200 y pasillo vacío", code, acc.Estado.Pasillo)
	}
	var episodios []episodioDTO
	pedir(t, s, "GET", "/api/doctores/D-001/episodios", "", &episodios)
	if len(episodios) != 2 || episodios[1].Habitacion != 101 {
		t.Errorf("Karen debía ver 2 episodios, el segundo ya con la 101: %+v", episodios)
	}
}

func TestErroresHTTP(t *testing.T) {
	s := nuevoServidorPrueba()
	casos := []struct {
		metodo, ruta, cuerpo string
		want                 int
	}{
		{"POST", "/api/pacientes/P-999/ataque", "", http.StatusNotFound},
		{"GET", "/api/doctores/D-999/episodios", "", http.StatusNotFound},
		{"POST", "/api/pacientes/P-001/despertar", "", http.StatusConflict}, // ya estaba despierto
		{"POST", "/api/pacientes/P-001/ataque", "{no es json", http.StatusBadRequest},
		{"GET", "/api/juego", "", http.StatusNotFound},
		{"POST", "/api/juego/nuevo", `{"bots":7}`, http.StatusBadRequest},
		{"DELETE", "/api/estado", "", http.StatusMethodNotAllowed},
	}
	for _, c := range casos {
		t.Run(c.metodo+" "+c.ruta, func(t *testing.T) {
			var e errorDTO
			if code := pedir(t, s, c.metodo, c.ruta, c.cuerpo, nil); code != c.want {
				t.Errorf("código = %d, se esperaba %d", code, c.want)
			}
			if c.want != http.StatusMethodNotAllowed {
				pedir(t, s, c.metodo, c.ruta, c.cuerpo, &e)
				if e.Error == "" {
					t.Errorf("la respuesta debía traer un mensaje de error")
				}
			}
		})
	}
}

func TestSeverosYReiniciar(t *testing.T) {
	s := nuevoServidorPrueba()
	pedir(t, s, "POST", "/api/pacientes/P-002/ataque", "", nil)
	var severos []pacienteDTO
	pedir(t, s, "GET", "/api/severos", "", &severos)
	if len(severos) != 3 || severos[1].ID != "P-002" || severos[1].Episodios != 1 {
		t.Fatalf("severos ordenados con P-002 en 1 episodio; llegó %+v", severos)
	}

	var acc accionDTO
	pedir(t, s, "POST", "/api/escenario/reiniciar", "", &acc)

	if len(acc.Estado.Historial) != 0 {
		t.Errorf("después de reiniciar el historial debía estar vacío")
	}
}

func TestJuegoCompletoPorAPI(t *testing.T) {
	s := nuevoServidorPrueba()
	var juego juegoDTO
	if code := pedir(t, s, "POST", "/api/juego/nuevo", `{"nombre":"Sofía","bots":2}`, &juego); code != http.StatusOK {
		t.Fatalf("nuevo: código %d", code)
	}
	if len(juego.Jugadores) != 3 || len(juego.Mano) != 3 || juego.Turno != 0 {
		t.Fatalf("partida mal repartida: %+v", juego)
	}
	// Sofía descarta siempre; los bots juegan solos hasta que alguien gana.
	for i := 0; i < 500 && juego.Ganador == ""; i++ {
		var siguiente juegoDTO
		code := pedir(t, s, "POST", "/api/juego/descartar", `{"cartas":[0]}`, &siguiente)
		if code != http.StatusOK {
			t.Fatalf("descartar en la vuelta %d: código %d", i, code)
		}
		if siguiente.Ganador == "" && siguiente.Turno != 0 {
			t.Fatalf("después de la jugada debía volver a tocarle a Sofía, toca a %d", siguiente.Turno)
		}
		juego = siguiente
	}
	if juego.Ganador == "" {
		t.Fatal("la partida no terminó")
	}
	if code := pedir(t, s, "POST", "/api/juego/jugar", `{"carta":0}`, nil); code != http.StatusConflict {
		t.Errorf("jugar después de terminar: código %d, se esperaba 409", code)
	}
}

func TestSirveElFrontend(t *testing.T) {
	s := nuevoServidorPrueba()
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Hospital")) {
		t.Errorf("GET / debía servir index.html; código %d", rec.Code)
	}
}

func TestPeticionesConcurrentes(t *testing.T) {
	s := nuevoServidorPrueba()
	ids := []string{"P-001", "P-002", "P-003", "P-004", "P-005"}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := ids[i%len(ids)]
			switch i % 3 {
			case 0:
				pedir(t, s, "POST", "/api/pacientes/"+id+"/ataque", "", nil)
			case 1:
				pedir(t, s, "POST", "/api/pacientes/"+id+"/despertar", "", nil)
			default:
				pedir(t, s, "GET", "/api/estado", "", nil)
			}
		}()
	}
	wg.Wait()

	var estado estadoDTO
	pedir(t, s, "GET", "/api/estado", "", &estado)
	for _, hab := range estado.Habitaciones {
		if len(hab.Ocupantes) > hab.Capacidad {
			t.Errorf("habitación %d con %d ocupantes", hab.Numero, len(hab.Ocupantes))
		}
	}
}
