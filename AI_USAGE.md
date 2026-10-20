# Uso de IA en el proyecto

Este archivo cuenta de forma honesta cómo se usó la inteligencia artificial. El registro detallado,
entrada por entrada, está en `docs/bitacora_ia.md`.

## Herramientas y para qué

| Herramienta | Para qué se usó |
|---|---|
| **Claude Code** (modelo Claude Opus 5.5, en la terminal) | Leer el enunciado, hacerme preguntas de diseño antes de cada etapa, escribir el código de todos los paquetes, los tests y la documentación, correr las verificaciones (`gofmt`, `go vet`, `go test`, `-race`) y hacer los commits. |
| **Claude in Chrome** | Que Claude probara la página web en el navegador: recorrer el escenario, jugar una partida, revisar la consola y el ancho de celular. |
| **Docker** (imagen oficial `golang:1.22`) | No es IA: se usó para correr `-race`, porque mi portátil con Windows no tiene compilador de C (cgo). |
| **`CLAUDE.md`** | Documento de instrucciones que Claude Code lee al empezar cada sesión: reglas del trabajo, ambientación, etapas y qué preguntarme. [COMPLETAR: cómo lo escribiste tú (a mano, con ayuda de otra IA, etc.)] |

**Cómo se repartió el trabajo.** La gran mayoría del código la escribió Claude. Mi papel fue decidir:
en cada etapa Claude me hizo preguntas con opciones (cómo despachar al personal, qué pasa con el episodio
del paciente del pasillo, si el camillero embebe `Persona`, el estilo de la web, etc.) y yo elegí. Cuando no
entendí una opción, le pedí que me explicara las alternativas antes de decidir. Al final de cada etapa Claude
me explicó lo que hizo y qué me podían preguntar en la defensa.
[COMPLETAR: qué partes revisaste o modificaste tú directamente, y cómo te preparaste para explicar cada línea]

## Prompts decisivos

1. **"Lee CLAUDE.md y docs/enunciado.pdf. Empecemos por la etapa 0: hazme las preguntas que necesites antes
   de crear nada."**
   Fijó la forma de trabajar de todo el proyecto: primero preguntas y propuesta, después código, con un
   commit por parte. En esa primera ronda decidí hacer el plan completo (núcleo, simulación, mini-juego y web)
   aunque la entrega era el mismo día, y usar el módulo `costenos-narcolepsia`.

2. **"¿Cómo lo recomiendas y por qué? ¿Qué tiene un método sobre el otro?"**
   Lo pregunté cuando Claude me propuso dos formas de compartir el registro del episodio entre el hospital y
   el doctor. Me explicó las ventajas y desventajas de cada una y elegí la interfaz opcional privada
   `llevaHistorial` en lugar de agregar un método exportado `AnotarEpisodio` a `Atendedor`: la interfaz queda
   más pequeña, el camillero no tiene que implementar un método vacío y nadie de afuera puede meterle
   episodios falsos a un doctor.

3. **"Necesito que elimines a Jaime de todo el proyecto. Cámbialo por 'El Coste', que cumpla el mismo rol,
   pero que no se llame Jaime."**
   Cambió el villano en 22 archivos (textos, identificadores como `CartaCoste`, tests, la imagen de la web y
   `CLAUDE.md`), cuidando la gramática ("al Coste", "del Coste").

## Casos en que el código generado estaba mal o no era idiomático

1. **Estado final inconsistente en la simulación.** En una corrida de `go run . -simulacion`, una paciente
   terminó en el pasillo con dos habitaciones libres, porque la simulación se cortó entre dos rondas del
   pasillo (no era una data race). Se agregó un "cierre de turno": cuando terminan todas las goroutines se
   hace una última ronda. El test de la simulación ahora verifica que nadie quede en el pasillo con camas
   libres.

2. **Una carta ASCII perdía información sin avisar.** La carta de Dayana salía sin su equipo: el texto
   "AMARILLO · JAGUARES DE CÓRDOBA" no cabía junto al ID y la función `izqDer` descartaba la parte derecha.
   Se acortaron los nombres de los equipos en las cartas.

3. **Código que no compilaba.** Al agregar el flag `-web`, el script con el que Claude editó `main.go`
   metió un salto de línea real dentro del string de `fmt.Printf` en lugar de `\n`. Se corrigió a mano.

4. **Código no idiomático.**
   - En un test los IDs se armaban con `string(rune('A'+i))`; se cambió por `fmt.Sprintf("P-%03d", i)`.
   - En `api/dto.go` la especialidad de cada doctor se copiaba con dos ciclos anidados; se cambió por un
     mapa ID -> especialidad.
   - `gofmt -l .` marcó la alineación de varios structs (`Config` y `Resumen` de `simulacion`, `datos` en
     `main.go`); se corrigieron con `gofmt -w`.

5. **Error visual detectado probando en Chrome.** Cada vez que la página se repintaba, todas las cartas
   repetían la animación de voltear, incluso al solo seleccionar una. Se agregó un "animador" que solo
   anima las cartas nuevas o que cambiaron de estado.

6. **Una trampa de embedding que se evitó.** Se pensó hacer que `CartaDoctor` embebiera `CartaCama` para
   heredar su método `Jugar`, pero el método promovido recibe la `CartaCama` interna, así que esa (y no el
   doctor) habría quedado encima del paciente. Se usó una función común `jugarCama(j, obj, carta)` que
   recibe la carta concreta.

Además, para entender la concurrencia, en una copia del proyecto se le quitó el mutex a `AsignarHabitacion`
y se corrió el test de 50 goroutines: hubo hasta 16 asignaciones con solo 6 camas y el detector reportó
9 data races. Con el mutex, el resultado es exacto.