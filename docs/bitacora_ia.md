# Bitácora de uso de IA

Registro honesto de cómo se usó Claude Code en el proyecto. Alimenta `AI_USAGE.md`.

Cada entrada lleva la fecha y uno de estos tipos:
- **Decisión**: un prompt o una decisión de Sofía que cambió el rumbo.
- **Corrección**: código generado que estaba mal, no compilaba, no pasaba `go vet` o `-race`,
  o no era idiomático, y cómo se arregló.

---

## 2026-10-19 — Decisión: alcance y configuración inicial (etapa 0)

- Sofía pidió leer `CLAUDE.md` y `docs/enunciado.pdf` y hacerle preguntas antes de crear nada.
- Claude señaló que el PDF fija la entrega para el 19/10/2026 23:59 y la defensa para el 20/10.
  Sofía decidió hacer el **plan completo** (núcleo, simulación, mini-juego y web) con commits por partes.
- Módulo: `costenos-narcolepsia` (el que sugiere el PDF) en lugar de `hospitalgo`; se actualizó `CLAUDE.md`.
- Consola con colores ANSI, desactivables con `-sin-color` o `NO_COLOR`.
- Se mantienen los nombres, personajes y chistes de `CLAUDE.md`.

## 2026-10-19 — Decisión: diseño del núcleo (etapa 1)

- Despacho por turno rotativo; se agrega `Disponible() bool` a `Atendedor`
  (doctor disponible con menos de 3 pacientes a cargo; camillero siempre).
- Si el paciente del pasillo recibe cama después, se actualiza su mismo episodio (no se crea otro),
  para no contar doble en `ReporteSeveros`.
- `Camillero` embebe `Persona`. Despertar a alguien despierto devuelve `ErrPacienteDespierto`.
- Sofía pidió que Claude le explicara las dos opciones para compartir el registro entre hospital y doctor.
  Eligió la **interfaz opcional** privada `llevaHistorial` (type assertion a interfaz, no a tipo concreto)
  en lugar de agregar `AnotarEpisodio` exportado a `Atendedor`: interfaz más pequeña, el camillero no
  implementa un método vacío y nadie de afuera puede inyectar episodios.
- El equipo de cada paciente (color en consola) vive en `presentacion`, no en el modelo.

## 2026-10-19 — Correcciones durante la etapa 1

- **Corrección (test no idiomático):** en `TestDoctorSinCupoNoSeDespacha` Claude generó IDs con
  `string(rune('A'+i))`, que compila pero es difícil de leer. Se cambió por `fmt.Sprintf("P-%03d", i)`.
- **Corrección (carta truncada):** la carta de Dayana (P-004) salía sin equipo: "AMARILLO · JAGUARES DE
  CÓRDOBA" no cabía junto al ID y `izqDer` descartaba el texto de la derecha sin avisar. Se acortaron los
  nombres de equipo en la carta ("Jaguares", "Selección").
- **Corrección (dato confuso):** después de despertar, la carta de Wilfrido decía "lugar: habitación 101"
  mientras Edinson ya ocupaba esa habitación. Para pacientes despiertos la carta ahora dice "último lugar:".
- **Corrección (herramienta):** un script de shell que escribía varios archivos a la vez no se pudo
  interpretar (comillas sin cerrar) y no escribió nada; los archivos se crearon uno por uno.

## 2026-10-19 — Etapa 2: verificación de -race y experimento sin mutex

- **Decisión:** el portátil de Sofía no tiene compilador de C (`-race` exige cgo en Windows). Sofía eligió
  verificar `-race` con Docker (imagen oficial `golang:1.22`, que además comprueba la versión mínima de `go.mod`).
- **Experimento:** en una copia temporal se quitó el lock de `AsignarHabitacion` y se corrió
  `TestAsignacionConcurrenteNuncaSuperaCapacidad` 3 veces: hasta 16 asignaciones con 6 camas, la
  habitación 102 con 4 ocupantes y capacidad 2, y 9 `WARNING: DATA RACE` en `Habitacion.Ocupar`.
  Con el mutex: 6 asignaciones exactas y sin advertencias.

## 2026-10-19 — Correcciones durante la etapa 2

- **Corrección (gofmt):** Claude escribió los structs `Config` y `Resumen` de `simulacion` y el struct
  `datos` de `main.go` con los comentarios y campos mal alineados; `gofmt -l .` los marcó y se corrigieron
  con `gofmt -w`.
- **Corrección (estado final inconsistente):** en una corrida de `-simulacion`, Yeimy terminó en el pasillo
  con las habitaciones 101 y 103 libres, porque la simulación se cortó entre dos rondas del pasillo
  (no era data race). Se agregó un "cierre de turno": cuando terminan las goroutines, una última ronda.
  El test de la simulación ahora verifica que nadie quede en el pasillo con camas libres.

## 2026-10-19 — Etapa 3: decisiones y correcciones

- **Decisión:** mazo de 68 cartas con la proporción del Virus! original (Doctor y Camillero en lugar de
  las medicinas multicolor); bots La Seño Eneida, El Compa Jhonier y Tía Nury con frases que Sofía
  aprobó antes de meterlas; sin modo de partida rápida.
- **Corrección (orden de mensajes):** en `JugarTurnoBot` la frase del bot ("X jugó Y") se anotaba después
  de los mensajes que generaba la jugada, así que "¡Se fue pa' Barranquilla!" salía antes que la jugada.
  Se inserta la frase en la posición anterior a la jugada (`slices.Insert`).
- **Diseño evitado (trampa de embedding):** se consideró que `CartaDoctor` embebiera `CartaCama` para
  heredar `Jugar`, pero el método promovido recibiría la `CartaCama` interna y esa (no el doctor) quedaría
  encima del paciente y luego en el descarte. Se usa una función común `jugarCama(j, obj, carta)` que
  recibe la carta concreta.

## 2026-10-19 — Etapa 4: decisiones y correcciones

- **Decisión:** estilo carnaval de Barranquilla, sonidos sintetizados con Web Audio (con botón de silencio),
  los mismos bots del paquete `virus`. `main` inyecta en la API la función que arma el hospital para no
  duplicar el escenario. `hospital/` no cambió en esta etapa (`git diff ae923f2 -- hospital/` vacío).
- **Corrección (no idiomático):** en `api/dto.go` la especialidad de cada doctor se copiaba al personal con
  dos ciclos anidados; se reemplazó por un mapa ID -> especialidad.
- **Corrección (no compilaba):** al agregar el flag `-web`, el script que editó `main.go` metió un salto de
  línea real dentro del string de `fmt.Printf` en lugar de `\n`; se corrigió a mano.
- **Corrección (visual, detectada probando en Chrome):** cada vez que se repintaba la página, todas las
  cartas repetían la animación de voltear, incluso al solo seleccionar una carta. Se agregó
  `crearAnimador` en `comun.js` para animar solo las cartas nuevas o que cambiaron de estado.
- **Verificación en navegador:** se recorrió el escenario en la web (4 ataques, aviso sin cama, despertar,
  buscar cama con el chiste de Barranquilla, las 4 consultas) y una jugada válida e inválida del juego;
  sin errores en consola y sin scroll horizontal a 375 px de ancho.

## 2026-10-19 — Decisión: el villano pasa a llamarse "El Coste"

- Sofía pidió eliminar a "Jaime" de todo el proyecto y reemplazarlo por **El Coste**, con el mismo rol
  (Paciente Cero en el escenario y carta especial del mini-juego).
- Se cambiaron 86 apariciones en 22 archivos: textos, errores, identificadores (`CartaCoste`,
  `ErrCosteSinVictimas`, `frasesCoste`, `CulpaCoste`), tests, web (`img/coste.svg`) y `CLAUDE.md`,
  ajustando la gramática ("al Coste", "del Coste", "por el Coste").
- Esto tocó `hospital/` (solo el texto de `ErrNoEsOcupante` y el nombre de un paciente de prueba). No tiene
  que ver con la interfaz gráfica: la web sigue sin necesitar ningún cambio en el núcleo.

## 2026-10-19 — Etapa 5: documentación

- Se completó `README.md` y se armó `AI_USAGE.md` solo con lo registrado en esta bitácora. Lo que Sofía
  debe escribir con sus palabras quedó marcado `[COMPLETAR]`.
- **Corrección (documentación):** el primer diagrama de paquetes del README, dibujado con flechas ASCII, era
  confuso y no mostraba con certeza cada dependencia. Se reemplazó por una lista sacada de `go list`.
