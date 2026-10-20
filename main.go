// Programa principal del Hospital de los Costeños con Narcolepsia.
// Solo arma los datos, elige el modo según los flags e imprime; toda la
// lógica de negocio vive en el paquete hospital.
//
//	go run .                 escenario de demostración (sección 6 del enunciado)
//	go run . -simulacion     simulación concurrente (bono)
//	go run . -jugar          mini-juego "Virus del Sueño" en consola
//	go run . -web            API REST y página web en http://localhost:8080
//	go run . -sin-color      sin colores ANSI (también con NO_COLOR)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"costenos-narcolepsia/api"
	"costenos-narcolepsia/hospital"
	"costenos-narcolepsia/presentacion"
	"costenos-narcolepsia/simulacion"
	"costenos-narcolepsia/web"
)

// datos agrupa el hospital armado y las piezas que el escenario nombra.
type datos struct {
	h                                      *hospital.Hospital
	wilfrido, yeimy, jhon, dayana, edinson *hospital.Paciente
}

// lugares son los sitios donde les da el ataque en la simulación.
var lugares = []string{
	"en la mecedora del pórtico",
	"frente al picó en la verbena",
	"en la tienda de la esquina viendo al Junior",
	"en el Transmetro",
	"en la fila de la cafetería esperando el sancocho",
	"en la sala de espera de radiología",
	"en la terraza a la hora de la siesta",
}

func main() {
	sinColor := flag.Bool("sin-color", false, "desactiva los colores ANSI (también con la variable NO_COLOR)")
	simular := flag.Bool("simulacion", false, "corre la simulación concurrente en vez del escenario")
	duracion := flag.Duration("duracion", 5*time.Second, "duración de la simulación, por ejemplo 10s")
	jugar := flag.Bool("jugar", false, "juega Virus del Sueño en la consola")
	servirWeb := flag.Bool("web", false, "arranca la API REST y la página web")
	direccion := flag.String("direccion", ":8080", "dirección donde escucha el servidor web")
	flag.Parse()
	log.SetFlags(0)

	d := armarHospital()
	consola := presentacion.NuevaConsola(!*sinColor && os.Getenv("NO_COLOR") == "", equipos(d))

	if *servirWeb {
		correrWeb(*direccion)
		return
	}
	if *jugar {
		jugarVirus(consola)
		return
	}
	if *simular {
		correrSimulacion(d, consola, *duracion)
		return
	}
	correrEscenario(d, consola)
}

// armarHospital crea el hospital del paso 1: 2 doctores, 1 camillero,
// 3 habitaciones de capacidad 1 y 5 pacientes (3 Severos).
func armarHospital() datos {
	h := hospital.NuevoHospital("Hospital de los Costeños con Narcolepsia")
	debe(h.ContratarDoctor(hospital.NuevoDoctor("D-001", "Dra. Karen Ospina", 43, "Neurología del sueño")))
	debe(h.ContratarDoctor(hospital.NuevoDoctor("D-002", `Dr. Rafael "Rafa" Escalona Pérez`, 51, "Medicina interna y vallenato")))
	debe(h.ContratarPersonal(hospital.NuevoCamillero("C-001", `Kevin "Piernas" Charris`, 24)))
	for _, numero := range []int{101, 102, 103} {
		hab, err := hospital.NuevaHabitacion(numero, 1)
		debe(err)
		debe(h.AgregarHabitacion(hab))
	}

	d := datos{
		h:        h,
		wilfrido: hospital.NuevoPaciente("P-001", `Wilfrido Berrío "el Cachaco de Soledad"`, 58, hospital.Severo),
		yeimy:    hospital.NuevoPaciente("P-002", "Yeimy Padilla", 34, hospital.Severo),
		jhon:     hospital.NuevoPaciente("P-003", `Jhon Jairo "Mi Llave" Mercado`, 41, hospital.Moderado),
		dayana:   hospital.NuevoPaciente("P-004", "Dayana Barraza", 27, hospital.Leve),
		edinson:  hospital.NuevoPaciente("P-005", `Edinson "el Bacán" Cantillo`, 30, hospital.Severo),
	}
	for _, p := range []*hospital.Paciente{d.wilfrido, d.yeimy, d.jhon, d.dayana, d.edinson} {
		debe(h.AdmitirPaciente(p))
	}
	return d
}

// equipos asigna a cada paciente del escenario el equipo del que es hincha.
func equipos(d datos) map[string]presentacion.Equipo {
	return map[string]presentacion.Equipo{
		d.wilfrido.ID(): presentacion.Junior,
		d.yeimy.ID():    presentacion.UnionMagdalena,
		d.jhon.ID():     presentacion.RealCartagena,
		d.dayana.ID():   presentacion.Jaguares,
		d.edinson.ID():  presentacion.Junior,
	}
}

// correrWeb arranca el servidor. La API recibe la función que arma el
// hospital del escenario, para poder reiniciarlo sin duplicar los datos.
func correrWeb(direccion string) {
	servidor := api.NuevoServidor(api.Config{
		ArmarHospital: func() (*hospital.Hospital, map[string]string) {
			d := armarHospital()
			colores := make(map[string]string)
			for id, eq := range equipos(d) {
				colores[id] = eq.Color()
			}
			return d.h, colores
		},
		Lugares:  lugares,
		Archivos: web.Archivos,
	})
	fmt.Printf("Hospital de los Costeños en http://localhost%s (Ctrl+C para cerrar)\n", direccion)
	log.Fatal(http.ListenAndServe(direccion, servidor))
}

// correrSimulacion lanza la simulación concurrente, imprime cada evento en
// vivo y al final las cuatro consultas.
func correrSimulacion(d datos, c *presentacion.Consola, duracion time.Duration) {
	fmt.Println(c.Titulo(fmt.Sprintf("Simulación concurrente: %v de epidemia de sueño", duracion)))
	fmt.Println(c.Nota("Cada paciente es una goroutine que se duerme y se despierta al azar; una más hace la ronda del pasillo. El Coste anda suelto."))
	fmt.Println()

	resumen := simulacion.Ejecutar(context.Background(), d.h, simulacion.Config{
		Duracion:    duracion,
		Ubicaciones: lugares,
	}, func(ev simulacion.Evento) {
		fmt.Println(c.LineaEvento(ev))
	})

	fmt.Println(c.Titulo("Resumen de la simulación"))
	fmt.Println(c.ResumenSimulacion(resumen))
	imprimirConsultas(d.h, c, true, []string{
		fmt.Sprintf("  Durante la simulación: %d camas al dormirse, %d por la ronda del pasillo y %d veces sin cama.",
			resumen.ConCama, resumen.LlevadosPorRonda, resumen.EnPasillo),
	})
}

// debe termina el programa si falla un paso del armado, que con estos datos
// fijos no debería fallar nunca.
func debe(err error) {
	if err != nil {
		log.Fatalf("error armando el escenario: %v", err)
	}
}
