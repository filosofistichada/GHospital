# Hospital de los Costeños con Narcolepsia — edición "Virus del Sueño"

**Autora:** Sofía [COMPLETAR: apellido] (trabajo individual)
**Curso:** Programación Orientada a Objetos — Universidad EIA, profesor Sebastián Zapata Ramírez

Sistema de gestión de un hospital costeño cuyos pacientes se quedan dormidos en cualquier parte,
escrito en Go con solo la librería estándar. El modelo usa structs con campos no exportados,
métodos con receptor, embedding e interfaces. Además del escenario obligatorio incluye los dos bonos
(simulación concurrente e interfaz web) y un mini-juego de cartas inspirado en *Virus!*.

El villano de la historia es **El Coste**, "el Paciente Cero": donde pasa, la gente se duerme.

## Requisitos

- Go 1.22 o superior (el módulo declara `go 1.22`). No hay dependencias externas.

## Cómo correr

```sh
go run .                           # escenario de demostración (sección 6 del enunciado)
go run . -sin-color                # sin colores ANSI (también con NO_COLOR=1)
go run . -simulacion               # simulación concurrente de 5 s (bono)
go run . -simulacion -duracion=10s # con otra duración
go run . -jugar                    # mini-juego "Virus del Sueño" contra 1 a 3 bots
go run . -web                      # API REST + página web en http://localhost:8080 (bono)
go run . -web -direccion=:9000     # en otro puerto
```

El escenario hace, en este orden: crea el hospital con 2 doctores, 1 camillero, 3 habitaciones de
capacidad 1 y 5 pacientes (3 Severos); cuatro pacientes se duermen y el cuarto se queda sin cama
(se imprime el error); uno despierta, libera su habitación y el del pasillo la recibe; y se imprimen
las cuatro consultas.

## Cómo probar

```sh
go test ./...
go test -race ./...   # detector de data races (requiere cgo)
gofmt -l .            # debe salir vacío
go vet ./...          # no debe reportar nada
```

En Windows, `-race` necesita un compilador de C (cgo). Sin él, se puede verificar con Docker
usando la imagen oficial de Go (desde la raíz del repositorio, en PowerShell):

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.22 go test -race ./...
docker run --rm -v "${PWD}:/src" -w /src golang:1.22 go run -race . -simulacion
```

## Estructura

Dependencias entre paquetes del proyecto (quién importa a quién, según `go list`):

```
main          -> api, hospital, presentacion, simulacion, virus, web
presentacion  -> hospital, simulacion, virus
simulacion    -> hospital
api           -> hospital, virus
hospital      -> (nada del proyecto)
virus         -> (nada del proyecto)
web           -> (nada del proyecto; solo embebe archivos)
```

| Paquete | Responsabilidad |
|---|---|
| `hospital/` | Modelo de objetos (núcleo obligatorio). No importa nada del proyecto y no tiene código de vista |
| `presentacion/` | Cartas ASCII y colores para la consola. Solo formatea, nunca decide reglas |
| `simulacion/` | Simulación concurrente: una goroutine por paciente y una ronda del pasillo |
| `virus/` | Mini-juego "Virus del Sueño". Independiente de `hospital` |
| `api/` | API REST con `net/http` y DTO propios; importa `hospital` y `virus` |
| `web/` | Frontend HTML/CSS/JS sin frameworks, embebido en el binario con `embed` |
| `main` | `main.go` (flags y armado del hospital), `escenario.go` (pasos del escenario), `jugar.go` (consola del juego) |

## Modelo (paquete `hospital`)

| Tipo | Qué es |
|---|---|
| `Persona` | Identidad (`id`, `nombre`, `edad`), **embebida por valor** en `Paciente`, `Doctor` y `Camillero` |
| `Paciente` | Nivel (`Leve`/`Moderado`/`Severo`), estado (`Despierto`/`DormidoEnPasillo`/`DormidoEnCama`), ubicación y doctor asignado |
| `Doctor` | Especialidad, pacientes a cargo y episodios atendidos; diagnostica |
| `Camillero` | Lleva pacientes a cama, pero no diagnostica |
| `Habitacion` | Número, capacidad, estado (`Disponible`/`Ocupada`) y ocupantes |
| `RegistroEpisodio` | Quién se durmió, cuándo, dónde, quién lo atendió y en qué habitación quedó |
| `Hospital` | Coordina todo, despacha al personal y responde las consultas |
| `Atendedor` | Interfaz: `ID`, `Nombre`, `Disponible`, `Atender`. La implementan `*Doctor` y `*Camillero` |

Las cuatro consultas devuelven datos (no imprimen):

```go
func (h *Hospital) PacientesEnPasillo() []*Paciente
func (d *Doctor) MisEpisodios() []RegistroEpisodio
func (h *Hospital) AsignarHabitacion(p *Paciente) (*Habitacion, error)
func (h *Hospital) ReporteSeveros() map[*Paciente]int
```

## Decisiones de diseño

- **Encapsulamiento.** Todos los campos del modelo están en minúscula; desde otro paquete solo se leen y
  cambian por métodos que validan (por ejemplo, no se puede despertar a alguien despierto ni meter a dos
  en una cama de capacidad 1). Los cambios internos (`acostarEnCama`, `anotarEpisodio`) tampoco se exportan.
- **Persona embebida por valor**, no como `*Persona`: cada entidad es dueña de su identidad, nunca es nil y
  dos pacientes no pueden compartir (y cambiarse) el mismo nombre por accidente.
- **Despacho polimórfico.** El hospital guarda `[]Atendedor` y elige por turno rotativo al siguiente que
  esté `Disponible()` (un doctor, si tiene menos de 3 pacientes a cargo; el camillero, siempre), sin
  preguntar el tipo concreto. `var _ Atendedor = (*Doctor)(nil)` comprueba en compilación que los tipos
  cumplen la interfaz.
- **Historial compartido.** Hospital y doctor guardan el mismo `*RegistroEpisodio`. Para entregárselo al
  doctor, el hospital usa una interfaz opcional privada (`llevaHistorial`): pregunta "¿llevas historial?"
  y no "¿eres Doctor?". Así, cuando el paciente del pasillo recibe cama después, el mismo registro se
  actualiza y el doctor lo ve en `MisEpisodios`, que devuelve copias.
- **`ReporteSeveros` devuelve un `map[*Paciente]int`** porque el enunciado pide el emparejamiento
  paciente -> episodios del día; la clave es el puntero para que sea el mismo paciente del hospital.
  Incluye a los Severos con 0 episodios. Como un map no tiene orden, la presentación ordena por ID.
- **Errores.** Todo lo que puede fallar devuelve `error` como último valor; nunca hay `panic` en casos
  previsibles. Los errores son variables exportadas (`ErrSinHabitacion`, ...) envueltas con `%w`, así que
  se reconocen con `errors.Is`.
- **Concurrencia.** `Hospital` tiene un `sync.Mutex` que toman todos sus métodos exportados. Como el mutex
  no es reentrante, la lógica compartida vive en métodos internos con sufijo `Locked`. Los métodos de
  `Paciente`, `Doctor` y `Habitacion` no son seguros por sí solos: todo cambio pasa por el hospital.
  Sin el mutex, un test con 50 goroutines llegó a asignar 16 camas cuando había 6.
- **Separación vista/modelo.** La consola (`presentacion`) y la web (`api` + `web`) están fuera del
  núcleo. El equipo de cada paciente (color) es apariencia y vive en la presentación. La interfaz gráfica
  se agregó sin cambiar el paquete `hospital`: `main` le inyecta a la API la función que arma el hospital.
- **Mini-juego.** `Carta` es una interfaz con 11 tipos concretos que embeben `cartaBase`; el juego llama
  `carta.Jugar(...)` sin saber qué carta es. Cada carta valida antes de modificar nada, así que una jugada
  inválida deja el juego intacto (y los bots pueden probar jugadas). El azar se inyecta (`*rand.Rand`)
  para que los tests sean deterministas.

## Documentos

- `AI_USAGE.md`: uso de inteligencia artificial en el proyecto.
- `docs/bitacora_ia.md`: registro detallado de decisiones y correcciones durante el desarrollo.
- `web/img/CREDITOS.md`: origen y licencia de los recursos gráficos.
