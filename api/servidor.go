package api

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"costenos-narcolepsia/hospital"
	"costenos-narcolepsia/virus"
)

// Config es lo que el servidor recibe de main. La API no conoce los datos
// del escenario: main le pasa la función que arma el hospital, así el
// escenario vive en un solo lugar y "reiniciar" la vuelve a llamar.
type Config struct {
	// ArmarHospital crea el hospital del escenario y devuelve, por ID de
	// paciente, el color de su equipo ("rojo", "azul", ...).
	ArmarHospital func() (*hospital.Hospital, map[string]string)
	// Lugares son las ubicaciones al azar cuando un ataque no trae una.
	Lugares []string
	// Archivos es el frontend (HTML, CSS, JS, imágenes).
	Archivos fs.FS
	// NuevoAzar crea el generador de cada partida. Si es nil, usa la hora.
	NuevoAzar func() *rand.Rand
}

// Servidor atiende la API REST y sirve el frontend.
//
// mu serializa todas las peticiones: además de proteger los campos de
// abajo, garantiza que armar las respuestas leyendo pacientes y
// habitaciones no compita con otra petición que los esté cambiando.
type Servidor struct {
	cfg     Config
	mux     *http.ServeMux
	mu      sync.Mutex
	h       *hospital.Hospital
	equipos map[string]string
	juego   *virus.Juego
}

// NuevoServidor crea el servidor con el hospital recién armado.
func NuevoServidor(cfg Config) *Servidor {
	if cfg.NuevoAzar == nil {
		cfg.NuevoAzar = func() *rand.Rand {
			return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1))
		}
	}
	if len(cfg.Lugares) == 0 {
		cfg.Lugares = []string{"en el pasillo"}
	}
	s := &Servidor{cfg: cfg, mux: http.NewServeMux()}
	s.h, s.equipos = cfg.ArmarHospital()

	s.mux.HandleFunc("GET /api/estado", s.verEstado)
	s.mux.HandleFunc("GET /api/pasillo", s.verPasillo)
	s.mux.HandleFunc("GET /api/doctores/{id}/episodios", s.verEpisodiosDoctor)
	s.mux.HandleFunc("GET /api/severos", s.verSeveros)
	s.mux.HandleFunc("POST /api/pacientes/{id}/ataque", s.registrarAtaque)
	s.mux.HandleFunc("POST /api/pacientes/{id}/despertar", s.despertar)
	s.mux.HandleFunc("POST /api/pacientes/{id}/cama", s.asignarCama)
	s.mux.HandleFunc("POST /api/escenario/reiniciar", s.reiniciar)

	s.mux.HandleFunc("POST /api/juego/nuevo", s.nuevoJuego)
	s.mux.HandleFunc("GET /api/juego", s.verJuego)
	s.mux.HandleFunc("POST /api/juego/jugar", s.jugarCarta)
	s.mux.HandleFunc("POST /api/juego/descartar", s.descartar)

	if cfg.Archivos != nil {
		s.mux.Handle("GET /", http.FileServerFS(cfg.Archivos))
	}
	return s
}

// ServeHTTP implementa http.Handler.
func (s *Servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// escribirJSON responde con el valor codificado en JSON.
func escribirJSON(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}

// escribirError responde {"error": "..."} con el código dado.
func escribirError(w http.ResponseWriter, estado int, err error) {
	escribirJSON(w, estado, errorDTO{Error: err.Error()})
}

// leerJSON decodifica el cuerpo en v. Un cuerpo vacío no es error: deja v
// con sus valores por defecto.
func leerJSON(r *http.Request, v any) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
