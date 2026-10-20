// Página del hospital: dibuja el estado que manda la API y llama a sus
// endpoints. Todas las reglas (camas, despacho, errores) las decide Go.

import { api, el, avisar, iconoEquipo, sonido, prepararBotonSonido, crearAnimador } from "./comun.js";

let estado = null;
// Para el chiste de Barranquilla: quién liberó una cama hace un momento.
let ultimoDespertado = null;
const animador = crearAnimador();

async function cargar() {
  try {
    pintar(await api("GET", "/api/estado"));
  } catch (err) {
    avisar(`No se pudo cargar el hospital: ${err.message}`, "alerta", 8000);
  }
}

// accion llama a un endpoint POST y vuelve a pintar con el estado nuevo.
async function accion(ruta) {
  try {
    const respuesta = await api("POST", ruta);
    pintar(respuesta.estado);
    return respuesta;
  } catch (err) {
    sonido.error();
    avisar(err.message, "alerta");
    return null;
  }
}

async function dormir(p) {
  const r = await accion(`/api/pacientes/${p.id}/ataque`);
  if (!r) return;
  sonido.ronquido();
  sacudir(p.id);
  if (r.aviso) {
    avisar(r.aviso, "alerta", 8000);
    document.getElementById("ultima-asignacion").textContent = `Último intento con ${p.nombre}: ${r.aviso}`;
  } else {
    avisar(r.mensaje);
  }
}

async function despertar(p) {
  const r = await accion(`/api/pacientes/${p.id}/despertar`);
  if (!r) return;
  sonido.clic();
  avisar(r.mensaje);
  if (p.codigo === "cama") ultimoDespertado = p;
  if (p.codigo === "cama" && r.estado.pasillo.length > 0) {
    avisar(`Se liberó la habitación ${p.habitacion}. Hay gente en el pasillo: ¡búsquenle cama!`, "chiste");
  }
}

async function buscarCama(p) {
  const r = await accion(`/api/pacientes/${p.id}/cama`);
  if (!r) return;
  sonido.cama();
  const ahora = r.estado.pacientes.find((x) => x.id === p.id);
  document.getElementById("ultima-asignacion").textContent =
    `Último AsignarHabitacion: ${p.nombre} -> habitación ${ahora.habitacion}`;
  if (ultimoDespertado && ultimoDespertado.habitacion === ahora.habitacion) {
    sonido.barranquilla();
    avisar(`¡El que se fue pa' Barranquilla perdió su silla! ${nombreCorto(ultimoDespertado)} se despertó para ir a ver al Junior y ${nombreCorto(p)} ya está roncando en la habitación ${ahora.habitacion}.`, "chiste", 9000);
    ultimoDespertado = null;
  } else {
    avisar(r.mensaje);
  }
}

// ---------- Dibujo ----------

function pintar(nuevo) {
  estado = nuevo;
  const enHabitacion = new Map();
  for (const hab of estado.habitaciones) for (const id of hab.ocupantes) enHabitacion.set(id, hab.numero);

  rellenar("zona-espera", estado.pacientes.filter((p) => p.codigo === "despierto").map(cartaPaciente),
    "Nadie despierto. El sancocho hizo lo suyo.");
  rellenar("zona-pasillo", estado.pasillo.map(cartaPaciente), "Pasillo despejado.");

  rellenar("zona-habitaciones", estado.habitaciones.map((hab) => {
    const ocupantes = estado.pacientes.filter((p) => hab.ocupantes.includes(p.id));
    return el("div", { clase: `habitacion ${hab.disponible ? "libre" : ""}` },
      el("h3", {}, `Habitación ${hab.numero}`, el("span", { clase: "insignia" }, `${hab.estado} · ${hab.ocupantes.length}/${hab.capacidad}`)),
      ocupantes.length ? el("div", { clase: "fila-cartas" }, ocupantes.map(cartaPaciente)) : el("p", { clase: "vacio" }, "Cama tendida, esperando a alguien."));
  }));

  rellenar("zona-personal", estado.personal.map(cartaPersonal));
  pintarConsultas();
  animador.terminar();
}

function cartaPaciente(p) {
  const acciones = [];
  if (p.codigo === "despierto") {
    acciones.push(el("button", { type: "button", clase: "chico", onclick: () => dormir(p) }, "😴 Le dio el sueño"));
  } else {
    if (p.codigo === "pasillo") acciones.push(el("button", { type: "button", clase: "chico", onclick: () => buscarCama(p) }, "🛏️ Buscar cama"));
    acciones.push(el("button", { type: "button", clase: "chico secundario", onclick: () => despertar(p) }, "⏰ Despertar"));
  }

  const carta = el("article", { clase: `carta equipo-${p.equipo} ${p.codigo !== "despierto" ? "dormido" : ""}`, "data-id": p.id },
    el("div", { clase: "franja" }, el("img", { src: iconoEquipo[p.equipo], alt: "" }), `${p.id} · ${p.equipo}`),
    el("div", { clase: "cuerpo" },
      el("p", { clase: "nombre" }, p.nombre),
      el("p", { clase: "dato" }, `${p.edad} años `, el("span", { clase: `insignia ${p.nivel}` }, p.nivel)),
      el("p", { clase: "dato" }, `${p.estado}`),
      el("p", { clase: "dato" }, `${p.codigo === "despierto" ? "Último lugar" : "Lugar"}: ${p.ubicacion}`),
      el("p", { clase: "dato" }, `Doctor: ${p.doctor || "sin diagnóstico"}`),
      el("div", { clase: "acciones" }, acciones)));
  animador.marcar(carta, `${p.id}-${p.codigo}`);

  // Como en Virus!: el ataque o la cama se apilan encima de la carta.
  if (p.codigo === "despierto") return carta;
  const encima = p.codigo === "cama"
    ? el("div", { clase: "encima cama" }, `🛏️ Cama: habitación ${p.habitacion}`)
    : el("div", { clase: "encima ataque" }, `😴 Ataque de sueño ${p.ubicacion}`);
  return el("div", { clase: "pila" }, encima, carta);
}

function cartaPersonal(a) {
  const doctor = a.rol === "doctor";
  return animador.marcar(el("article", { clase: `carta ${doctor ? "equipo-azul" : "equipo-multicolor"}` },
    el("div", { clase: "franja" }, el("img", { src: doctor ? "img/doctor.svg" : "img/camillero.svg", alt: "" }), doctor ? "Doctor" : "Personal de apoyo"),
    el("div", { clase: "cuerpo" },
      el("p", { clase: "nombre" }, a.nombre),
      el("p", { clase: "dato" }, doctor ? a.especialidad : "Lleva a cama, pero no diagnostica"),
      el("p", { clase: "dato" }, el("span", { clase: "insignia" }, a.disponible ? "disponible" : "sin cupo")))), a.id);
}

function pintarConsultas() {
  rellenar("consulta-pasillo", estado.pasillo.map((p) =>
    el("li", { style: `--color: var(--${colorCss(p.equipo)})` }, el("span", { clase: "punto" }), `${p.id} ${p.nombre} (${p.ubicacion})`)),
  "Nadie dormido en el pasillo.", "li");

  rellenar("consulta-episodios", estado.doctores.map((d) =>
    el("div", {},
      el("p", { style: "margin:6px 0 2px;font-weight:800" }, `${d.nombre} (${d.episodios.length})`),
      d.episodios.length
        ? el("ul", {}, d.episodios.map((e) => el("li", { style: `--color: var(--${colorCss(e.equipo)})` },
          el("span", { clase: "punto" }), `${e.hora} ${e.pacienteId} ${e.ubicacion} `,
          el("strong", {}, e.habitacion ? `-> habitación ${e.habitacion}` : "-> pasillo"))))
        : el("p", { clase: "vacio" }, "Sin episodios todavía."))));

  rellenar("consulta-camas", estado.habitaciones.map((h) =>
    el("li", {}, `Habitación ${h.numero}: `, el("strong", {}, h.disponible ? "disponible" : "ocupada"), ` (${h.ocupantes.length}/${h.capacidad})`)));

  rellenar("consulta-severos", estado.severos.map((p) =>
    el("li", { style: `--color: var(--${colorCss(p.equipo)})` }, el("span", { clase: "punto" }),
      `${p.id} ${p.nombre}: ${p.episodiosHoy} episodio(s) hoy`,
      el("span", { clase: "barra", "aria-hidden": "true" }, Array.from({ length: p.episodiosHoy }, () => el("i"))))),
  "No hay pacientes severos.", "li");

  rellenar("historial", [...estado.historial].reverse().map((e) => el("li", {}, e.resumen)),
    "Todavía nadie se ha dormido. Raro, con este calor.", "li");
}

// ---------- Ayudas ----------

function rellenar(id, nodos, textoVacio, etiquetaVacio = "p") {
  const contenedor = document.getElementById(id);
  contenedor.replaceChildren(...(nodos.length || !textoVacio ? nodos : [el(etiquetaVacio, { clase: "vacio" }, textoVacio)]));
}

function sacudir(id) {
  const carta = document.querySelector(`[data-id="${id}"]`);
  carta?.classList.add("sacudir");
}

function colorCss(equipo) {
  return equipo === "multicolor" ? "morado" : equipo;
}

function nombreCorto(p) {
  return p.nombre.split(" ")[0];
}

document.getElementById("boton-reiniciar").addEventListener("click", async () => {
  const r = await accion("/api/escenario/reiniciar");
  if (r) {
    ultimoDespertado = null;
    document.getElementById("ultima-asignacion").textContent = "";
    avisar(r.mensaje);
  }
});
prepararBotonSonido();
cargar();
