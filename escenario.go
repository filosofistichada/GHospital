package main

import (
	"fmt"

	"costenos-narcolepsia/hospital"
	"costenos-narcolepsia/presentacion"
)

// correrEscenario ejecuta, en orden, los pasos 2 a 4 del escenario
// obligatorio sobre el hospital armado en el paso 1, e imprime cada uno.
func correrEscenario(d datos, c *presentacion.Consola) {
	h := d.h
	fmt.Println(c.Titulo(h.Nombre()))
	fmt.Println(c.Nota(`Paciente cero: el Coste, costeño cachón profesional. Donde pasa, la gente se duerme de puro aburrimiento con sus excusas. Su versión: "No es lo que parece, mi amor, yo estaba dormido."`))

	// ---- Paso 1: lo que se armó ----
	fmt.Println(c.Titulo("Paso 1: personal, habitaciones y pacientes"))
	for _, a := range h.Personal() {
		fmt.Printf("  Personal: %s (%s)\n", a.Nombre(), a.ID())
	}
	fmt.Printf("  Habitaciones: 101, 102 y 103, capacidad 1 cada una\n\n")
	fmt.Println(presentacion.EnFila(cartasPacientes(c, h.Pacientes()), 3))

	// ---- Paso 2: cuatro ataques de sueño; el cuarto se queda sin cama ----
	fmt.Println(c.Titulo("Paso 2: cuatro ataques de sueño"))
	ataques := []struct {
		paciente  *hospital.Paciente
		ubicacion string
	}{
		{d.wilfrido, "en la mecedora del pórtico, contagiado por el Coste"},
		{d.yeimy, "frente al picó en la verbena, contagiada por el Coste"},
		{d.jhon, "en la tienda de la esquina viendo al Junior"},
		{d.edinson, "en la fila de la cafetería esperando el sancocho"},
	}
	var cartas []string
	var errSinCama error
	for _, a := range ataques {
		if err := h.RegistrarEpisodio(a.paciente, a.ubicacion); err != nil {
			errSinCama = err
		}
		historial := h.Historial()
		cartas = append(cartas, c.CartaEpisodio(historial[len(historial)-1]))
	}
	fmt.Println(presentacion.EnFila(cartas, 3))
	if errSinCama != nil {
		fmt.Println()
		fmt.Println(c.Error(errSinCama))
		fmt.Println(c.Nota(`"Dormirse es cosa de mujeres", dijo Edinson. Lo encontraron roncando en la fila de la cafetería, con la baba en la camiseta del Junior.`))
	}
	fmt.Println("\n  En el pasillo ahora mismo:")
	fmt.Println(presentacion.EnFila(cartasPacientes(c, h.PacientesEnPasillo()), 3))

	// ---- Paso 3: uno despierta, libera la habitación y el del pasillo la recibe ----
	fmt.Println(c.Titulo("Paso 3: el que se fue pa' Barranquilla perdió su silla"))
	debe(h.DespertarPaciente(d.wilfrido))
	habEdinson, err := h.AsignarHabitacion(d.edinson)
	debe(err)
	fmt.Println(c.Nota(presentacion.ChisteBarranquilla("Wilfrido", "Edinson", habEdinson.Numero())))
	fmt.Println(presentacion.EnFila(cartasPacientes(c, []*hospital.Paciente{d.wilfrido, d.edinson}), 3))

	// ---- Paso 4: las cuatro consultas ----
	notas := []string{fmt.Sprintf("  AsignarHabitacion(%s) en el paso 3 -> habitación %d", d.edinson.ID(), habEdinson.Numero())}
	if errSinCama != nil {
		notas = append(notas, fmt.Sprintf("  En el paso 2, sin cama libre -> error: %v\n", errSinCama))
	}
	imprimirConsultas(h, c, false, notas)
}

// imprimirConsultas imprime las cuatro consultas del enunciado. En modo
// compacto los episodios salen como líneas (Resumen) en vez de cartas.
// notasAsignacion se imprimen como detalle de la consulta 3.
func imprimirConsultas(h *hospital.Hospital, c *presentacion.Consola, compacto bool, notasAsignacion []string) {
	fmt.Println(c.Titulo("Consulta 1: pacientes dormidos en el pasillo (PacientesEnPasillo)"))
	if enPasillo := h.PacientesEnPasillo(); len(enPasillo) == 0 {
		fmt.Println("  Nadie en el pasillo: todo el mundo consiguió cama o ya se despertó.")
	} else {
		fmt.Println(presentacion.EnFila(cartasPacientes(c, enPasillo), 3))
	}

	for _, doc := range h.Doctores() {
		fmt.Println(c.Titulo("Consulta 2: episodios atendidos por " + doc.Nombre() + " (MisEpisodios)"))
		episodios := doc.MisEpisodios()
		if len(episodios) == 0 {
			fmt.Println("  Sin episodios.")
			continue
		}
		if compacto {
			for _, r := range episodios {
				fmt.Println("  " + r.Resumen())
			}
			continue
		}
		var cartas []string
		for _, r := range episodios {
			cartas = append(cartas, c.CartaEpisodio(r))
		}
		fmt.Println(presentacion.EnFila(cartas, 3))
	}

	fmt.Println(c.Titulo("Consulta 3: asignación de habitaciones (AsignarHabitacion)"))
	for _, nota := range notasAsignacion {
		fmt.Println(nota)
	}
	var cartasHab []string
	for _, hab := range h.Habitaciones() {
		cartasHab = append(cartasHab, c.CartaHabitacion(hab))
	}
	fmt.Println(presentacion.EnFila(cartasHab, 3))

	fmt.Println(c.Titulo("Consulta 4: reporte de severos (ReporteSeveros)"))
	fmt.Println(c.ReporteSeveros(h.ReporteSeveros()))
}

// cartasPacientes dibuja la carta de cada paciente.
func cartasPacientes(c *presentacion.Consola, pacientes []*hospital.Paciente) []string {
	cartas := make([]string, 0, len(pacientes))
	for _, p := range pacientes {
		cartas = append(cartas, c.CartaPaciente(p))
	}
	return cartas
}
