package virus

// TamanoMazo es la cantidad de cartas del mazo completo.
const TamanoMazo = 68

// nuevoMazo arma las 68 cartas, en la proporción del Virus! original:
// 21 pacientes, 17 ataques, 20 camas (16 + 2 doctores + 2 camilleros),
// 1 El Coste y 9 tratamientos. Sin barajar.
func nuevoMazo() []Carta {
	var mazo []Carta
	repetir := func(n int, crear func() Carta) {
		for i := 0; i < n; i++ {
			mazo = append(mazo, crear())
		}
	}

	// Pacientes: 5 por color y 1 comodín.
	hinchas := map[Color]string{
		Rojo:     "Edinson, hincha del Junior",
		Azul:     "Yeimy, hincha del Unión",
		Verde:    "Jhon Jairo, hincha del Real Cartagena",
		Amarillo: "Yorladis, hincha de Jaguares",
	}
	for _, color := range coloresEnOrden[:4] {
		nombre := hinchas[color]
		repetir(5, func() Carta {
			return &CartaPaciente{cartaBase{nombre, "Ponlo en tu tablero. No puedes tener dos del mismo equipo.", color}}
		})
	}
	repetir(1, func() Carta {
		return &CartaPaciente{cartaBase{"Selección Colombia en el Metropolitano", "Paciente comodín: vale por el equipo que te falte.", Multicolor}}
	})

	// Ataques de sueño: 2 comidas x 2 copias por color, y 1 comodín.
	comidas := map[Color][2]string{
		Rojo:     {"Arepa de huevo de las 2 p.m.", "Butifarra con bollo de yuca"},
		Azul:     {"Sancocho de pescado", "Arroz con coco y bocachico"},
		Verde:    {"Carimañola", "Patacón con suero"},
		Amarillo: {"Mote de queso", "Mote de queso"},
	}
	for _, color := range coloresEnOrden[:4] {
		for _, nombre := range comidas[color] {
			repetir(2, func() Carta {
				return &CartaAtaque{cartaBase{nombre, "Duerme a un paciente rival de su color. Si ya dormía, se fue pa' Barranquilla.", color}}
			})
		}
	}
	repetir(1, func() Carta {
		return &CartaAtaque{cartaBase{"Kola Román bien fría", "Ataque comodín: duerme a un paciente rival de cualquier color.", Multicolor}}
	})

	// Camas: 2 remedios x 2 copias por color, 2 doctores y 2 camilleros.
	remedios := map[Color][2]string{
		Rojo:     {"Hamaca", "Tinto cargado"},
		Azul:     {"Cama del hospital", "Agua de panela con limón"},
		Verde:    {"Jugo de corozo", "Hamaca"},
		Amarillo: {"Tinto cargado", "Cama del hospital"},
	}
	for _, color := range coloresEnOrden[:4] {
		for _, nombre := range remedios[color] {
			repetir(2, func() Carta {
				return &CartaCama{cartaBase{nombre, "Va sobre tu paciente de su color: lo despierta o lo protege. Dos camas: estable.", color}}
			})
		}
	}
	mazo = append(mazo,
		&CartaDoctor{cartaBase{"Dra. Karen Ospina", "Cama de cualquier color. Sabe diagnosticar: despierta a un dormido.", Multicolor}},
		&CartaDoctor{cartaBase{`Dr. Rafa Escalona`, "Cama de cualquier color. Sabe diagnosticar: despierta a un dormido.", Multicolor}},
	)
	repetir(2, func() Carta {
		return &CartaCamillero{cartaBase{`Kevin "Piernas" Charris`, "Cama de cualquier color, pero solo a un paciente despierto: no diagnostica.", Multicolor}}
	})

	// El Coste y los tratamientos costeños.
	mazo = append(mazo, &CartaCoste{cartaBase{`El Coste, "el Paciente Cero"`, "Pone un ataque a todos los pacientes de un rival, menos a los estables.", Multicolor}})
	repetir(2, func() Carta {
		return &CartaCambioDeNovia{cartaBase{"Me cambié de novia", "Intercambia uno de tus pacientes con uno de un rival.", Multicolor}}
	})
	repetir(3, func() Carta {
		return &CartaLaMoza{cartaBase{"La moza", "Róbale un paciente a un rival.", Multicolor}}
	})
	repetir(2, func() Carta {
		return &CartaChisme{cartaBase{"Chisme de la vecina", "Pasa tus ataques a pacientes libres de los rivales.", Multicolor}}
	})
	repetir(1, func() Carta {
		return &CartaPico{cartaBase{"Picó a todo volumen", "Todos los rivales botan su mano y roban otra.", Multicolor}}
	})
	repetir(1, func() Carta {
		return &CartaCachos{cartaBase{"Te montaron los cachos", "Intercambia todo tu tablero con el de un rival (menos los estables).", Multicolor}}
	})
	return mazo
}
