// Página del juego: dibuja la partida y arma las jugadas con clics. Qué es
// válido lo decide Go (paquete virus); aquí solo se pregunta lo que cada
// carta declara en "requiere" y se muestra el resultado.

import { api, el, avisar, iconoEquipo, sonido, prepararBotonSonido, crearAnimador } from "./comun.js";

let juego = null;
// Jugada en construcción: carta elegida, pasos que faltan y objetivo.
let jugada = null;
// Modo descarte: índices elegidos.
let descartando = null;
const animador = crearAnimador();

const $ = (id) => document.getElementById(id);

// ---------- Acciones ----------

async function empezar(evento) {
  evento.preventDefault();
  const bots = Number(document.querySelector('input[name="bots"]:checked').value);
  try {
    const nuevo = await api("POST", "/api/juego/nuevo", { nombre: $("nombre").value, bots });
    $("inicio").hidden = true;
    $("mesa").hidden = false;
    $("victoria").hidden = true;
    actualizar(nuevo);
  } catch (err) {
    avisar(err.message, "alerta");
  }
}

async function enviar(ruta, cuerpo) {
  try {
    actualizar(await api("POST", ruta, cuerpo));
  } catch (err) {
    sonido.error();
    avisar(err.message, "alerta", 6000);
    reiniciarSeleccion();
    pintar();
  }
}

function elegirCarta(carta) {
  sonido.clic();
  if (descartando) {
    descartando.has(carta.indice) ? descartando.delete(carta.indice) : descartando.add(carta.indice);
    pintar();
    return;
  }
  // Los pasos salen de lo que la carta dice que necesita (Requisitos en Go).
  const pasos = [];
  if (carta.requiere.pacientePropio) pasos.push("propio");
  if (carta.requiere.pacienteRival) pasos.push("pacienteRival");
  else if (carta.requiere.rival) pasos.push("rival");
  jugada = { carta, pasos, objetivo: { carta: carta.indice, rival: 0, pacienteRival: "", pacientePropio: "" } };
  avanzar();
}

// avanzar envía la jugada si ya no faltan pasos; si faltan, los pide.
function avanzar() {
  if (jugada.pasos.length === 0) {
    const objetivo = jugada.objetivo;
    reiniciarSeleccion();
    enviar("/api/juego/jugar", objetivo);
    return;
  }
  pintar();
}

function elegirPropio(color) {
  if (jugada?.pasos[0] !== "propio") return;
  jugada.objetivo.pacientePropio = color;
  jugada.pasos.shift();
  avanzar();
}

function elegirPacienteRival(indice, color) {
  if (jugada?.pasos[0] !== "pacienteRival") return;
  jugada.objetivo.rival = indice;
  jugada.objetivo.pacienteRival = color;
  jugada.pasos.shift();
  avanzar();
}

function elegirRival(indice) {
  if (jugada?.pasos[0] !== "rival") return;
  jugada.objetivo.rival = indice;
  jugada.pasos.shift();
  avanzar();
}

function reiniciarSeleccion() {
  jugada = null;
  descartando = null;
}

// ---------- Actualizar y animar ----------

function actualizar(nuevo) {
  const dormidosAntes = dormidos(juego);
  juego = nuevo;
  reiniciarSeleccion();
  pintar();

  // Sacude los pacientes que se acaban de dormir.
  for (const clave of dormidos(juego)) {
    if (!dormidosAntes.has(clave)) document.querySelector(`[data-clave="${clave}"]`)?.classList.add("sacudir");
  }
  mostrarMensajes(juego.mensajes);
  if (juego.ganador) {
    setTimeout(() => {
      $("texto-victoria").textContent = `¡Ganó ${juego.ganador}!`;
      $("victoria").hidden = false;
      sonido.fanfarria();
    }, Math.min(juego.mensajes.length, 6) * 900 + 400);
  }
}

// mostrarMensajes muestra los mensajes de a uno, con su sonido.
function mostrarMensajes(mensajes) {
  mensajes.forEach((m, i) => {
    setTimeout(() => {
      let tipo = "info";
      if (/[Ee]l Coste(?![a-zñ])/.test(m)) { tipo = "coste"; sonido.coste(); }
      else if (m.includes("Barranquilla")) { tipo = "chiste"; sonido.barranquilla(); }
      else if (m.includes("cachos") || m.includes("moza") || m.includes("Chisme") || m.includes("picó")) { tipo = "chiste"; sonido.clic(); }
      avisar(m, tipo, 6000);
    }, i * 900);
  });
}

function dormidos(estado) {
  const claves = new Set();
  if (!estado) return claves;
  for (const j of estado.jugadores) for (const p of j.pacientes) if (p.dormido) claves.add(`${j.indice}-${p.color}`);
  return claves;
}

// ---------- Dibujo ----------

function pintar() {
  if (!juego) return;
  const paso = jugada?.pasos[0];
  const yo = juego.jugadores[0];

  $("rivales").replaceChildren(...juego.jugadores.slice(1).map((j) => tableroJugador(j, paso)));
  $("mi-tablero").replaceChildren(tableroJugador(yo, paso));
  $("mano").replaceChildren(...juego.mano.map(cartaMano));
  animador.terminar();

  $("instruccion").textContent = instruccion(paso);
  $("boton-descartar").hidden = !!descartando || !!jugada;
  $("boton-confirmar").hidden = !descartando;
  $("boton-confirmar").disabled = !descartando || descartando.size === 0;
  $("boton-cancelar").hidden = !descartando && !jugada;
}

function instruccion(paso) {
  if (juego.ganador) return `Partida terminada: ganó ${juego.ganador}.`;
  if (descartando) return "Elige de 1 a 3 cartas para botar.";
  if (!jugada) {
    if (juego.mano.length === 0) return "No tienes cartas: usa Descartar para pasar el turno.";
    return `Te toca. Elige una carta (quedan ${juego.mazo} en el mazo).`;
  }
  const nombre = jugada.carta.nombre;
  return {
    propio: `${nombre}: elige uno de TUS pacientes.`,
    pacienteRival: `${nombre}: elige un paciente de un rival.`,
    rival: `${nombre}: elige a qué rival (haz clic en su tablero).`,
  }[paso];
}

function tableroJugador(j, paso) {
  const esRival = j.indice !== 0;
  const elegible = esRival && paso === "rival";
  const zona = el("section", {
    clase: `zona jugador ${juego.turno === j.indice ? "de-turno" : ""} ${elegible ? "objetivo" : ""}`,
    onclick: elegible ? () => elegirRival(j.indice) : null,
    tabindex: elegible ? "0" : null,
    onkeydown: elegible ? (e) => (e.key === "Enter" || e.key === " ") && elegirRival(j.indice) : null,
  },
  el("h2", {}, j.nombre + (j.esBot ? " (bot)" : " (tú)"),
    el("span", { clase: "insignia sanos" }, `${j.sanos}/4 sanos · ${j.cartas} cartas`)),
  j.pacientes.length
    ? el("div", { clase: "fila-cartas" }, j.pacientes.map((p) => pacienteJuego(j, p, paso)))
    : el("p", { clase: "vacio" }, "Sin pacientes todavía."));
  return zona;
}

function pacienteJuego(j, p, paso) {
  const esMio = j.indice === 0;
  const elegible = (esMio && paso === "propio") || (!esMio && paso === "pacienteRival");
  const elegir = () => (esMio ? elegirPropio(p.color) : elegirPacienteRival(j.indice, p.color));
  let estadoTexto = "😃 sano";
  if (p.estable) estadoTexto = "👑 ESTABLE";
  else if (p.dormido) estadoTexto = "😴 dormido";
  else if (p.camas > 0) estadoTexto = "🛏️ sano, con cama";

  return animador.marcar(el("article", {
    clase: `carta mini equipo-${p.color} ${p.dormido ? "dormido" : ""} ${elegible ? "objetivo" : ""}`,
    "data-clave": `${j.indice}-${p.color}`,
    onclick: elegible ? (e) => { e.stopPropagation(); elegir(); } : null,
    tabindex: elegible ? "0" : null,
    role: elegible ? "button" : null,
    onkeydown: elegible ? (e) => (e.key === "Enter" || e.key === " ") && elegir() : null,
  },
  el("div", { clase: "franja" }, el("img", { src: iconoEquipo[p.color], alt: "" }), p.equipo),
  el("div", { clase: "cuerpo" },
    el("p", { clase: "nombre" }, p.nombre),
    el("p", { clase: "dato" }, estadoTexto),
    el("div", {},
      p.encima.map((n) => el("span", { clase: `chip ${p.dormido ? "ataque" : ""}` }, n)),
      p.culpaCoste ? el("span", { clase: "chip ataque" }, "culpa del Coste") : null))),
  `${j.indice}-${p.color}-${p.dormido}-${p.camas}`);
}

function cartaMano(c) {
  const elegida = jugada?.carta.indice === c.indice || descartando?.has(c.indice);
  const icono = c.icono.startsWith("equipo-") ? iconoEquipo[c.color] : `img/${c.icono}.svg`;
  return animador.marcar(el("button", {
    type: "button",
    clase: `carta de-mano equipo-${c.color} ${elegida ? "elegida" : ""}`,
    onclick: () => elegirCarta(c),
    disabled: !!juego.ganador || (!!jugada && !descartando),
    "aria-pressed": String(!!elegida),
  },
  el("div", { clase: `franja tipo-${c.tipo}` }, el("img", { src: icono, alt: "" }), c.tipo),
  el("div", { clase: "cuerpo" },
    el("p", { clase: "nombre" }, c.nombre),
    el("p", { clase: "dato" }, c.color),
    el("p", { clase: "descripcion" }, c.descripcion))), `${c.indice}-${c.nombre}`);
}

// ---------- Botones ----------

$("form-inicio").addEventListener("submit", empezar);
$("boton-descartar").addEventListener("click", () => {
  if (juego.mano.length === 0) {
    enviar("/api/juego/descartar", { cartas: [] });
    return;
  }
  descartando = new Set();
  pintar();
});
$("boton-confirmar").addEventListener("click", () => {
  const cartas = [...descartando];
  reiniciarSeleccion();
  enviar("/api/juego/descartar", { cartas });
});
$("boton-cancelar").addEventListener("click", () => {
  reiniciarSeleccion();
  pintar();
});
const volverAlInicio = () => {
  $("victoria").hidden = true;
  $("mesa").hidden = true;
  $("inicio").hidden = false;
  juego = null;
};
$("boton-nueva").addEventListener("click", volverAlInicio);
$("boton-otra").addEventListener("click", volverAlInicio);
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && (jugada || descartando)) {
    reiniciarSeleccion();
    pintar();
  }
});
prepararBotonSonido();

// Si ya había una partida (por ejemplo, al recargar), se retoma.
api("GET", "/api/juego").then((estado) => {
  if (estado.ganador) return;
  $("inicio").hidden = true;
  $("mesa").hidden = false;
  juego = estado;
  pintar();
}).catch(() => { /* no hay partida: se queda el formulario */ });
