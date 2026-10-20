// Utilidades compartidas por las dos páginas: llamadas a la API, avisos,
// sonidos y construcción de elementos. Aquí no hay reglas del hospital ni
// del juego: esas viven en Go.

// api llama a la API y devuelve el JSON. Si la respuesta es un error,
// lanza un Error con el mensaje que mandó el servidor.
export async function api(metodo, ruta, cuerpo) {
  const opciones = { method: metodo, headers: {} };
  if (cuerpo !== undefined) {
    opciones.headers["Content-Type"] = "application/json";
    opciones.body = JSON.stringify(cuerpo);
  }
  const respuesta = await fetch(ruta, opciones);
  const datos = await respuesta.json().catch(() => ({}));
  if (!respuesta.ok) {
    throw new Error(datos.error || `Error ${respuesta.status}`);
  }
  return datos;
}

// el crea un elemento. Los hijos de texto se agregan como texto (nunca como
// HTML), así un nombre con "<" no puede romper la página.
export function el(etiqueta, atributos = {}, ...hijos) {
  const nodo = document.createElement(etiqueta);
  for (const [clave, valor] of Object.entries(atributos)) {
    if (valor === false || valor === null || valor === undefined) continue;
    if (clave.startsWith("on")) nodo.addEventListener(clave.slice(2), valor);
    else if (clave === "clase") nodo.className = valor;
    else nodo.setAttribute(clave, valor === true ? "" : valor);
  }
  for (const hijo of hijos.flat()) {
    if (hijo === null || hijo === undefined || hijo === false) continue;
    nodo.append(hijo instanceof Node ? hijo : document.createTextNode(String(hijo)));
  }
  return nodo;
}

// crearAnimador recuerda qué cartas había en el dibujo anterior para que
// solo se volteen las nuevas (o las que cambiaron de clave).
export function crearAnimador() {
  let vistas = new Set();
  let actuales = new Set();
  return {
    marcar(nodo, clave) {
      actuales.add(clave);
      if (!vistas.has(clave)) nodo.classList.add("nueva");
      return nodo;
    },
    terminar() {
      vistas = actuales;
      actuales = new Set();
    },
  };
}

// iconoEquipo devuelve la imagen que representa a cada equipo (color).
export const iconoEquipo = {
  rojo: "img/tiburon.svg",
  azul: "img/ciclon.svg",
  verde: "img/castillo.svg",
  amarillo: "img/jaguar.svg",
  multicolor: "img/estrella.svg",
};

// avisar muestra un aviso flotante que se va solo.
// tipo: "info" | "alerta" | "chiste" | "coste".
export function avisar(texto, tipo = "info", duracion = 5000) {
  const contenedor = document.getElementById("avisos");
  const imagen = { alerta: "img/comida.svg", chiste: "img/tratamiento.svg", coste: "img/coste.svg" }[tipo];
  const aviso = el("div", { clase: `aviso ${tipo}`, role: "status" },
    imagen ? el("img", { src: imagen, alt: "" }) : null,
    el("p", { style: "margin:0" }, texto));
  contenedor.append(aviso);
  setTimeout(() => {
    aviso.classList.add("saliendo");
    aviso.addEventListener("animationend", () => aviso.remove(), { once: true });
    setTimeout(() => aviso.remove(), 600); // por si no hay animaciones
  }, duracion);
}

// ---------- Sonidos sintetizados con Web Audio (sin archivos) ----------

let contexto = null;
let silencio = leerSilencio();

function leerSilencio() {
  try {
    return localStorage.getItem("silencio") === "1";
  } catch {
    return false;
  }
}

// tono toca una nota: frecuencia inicial y final, duración y forma de onda.
function tono(desde, hasta, duracion, forma = "sine", retraso = 0, volumen = 0.15) {
  if (silencio) return;
  try {
    contexto ??= new AudioContext();
    const inicio = contexto.currentTime + retraso;
    const osc = contexto.createOscillator();
    const ganancia = contexto.createGain();
    osc.type = forma;
    osc.frequency.setValueAtTime(desde, inicio);
    osc.frequency.exponentialRampToValueAtTime(hasta, inicio + duracion);
    ganancia.gain.setValueAtTime(volumen, inicio);
    ganancia.gain.exponentialRampToValueAtTime(0.001, inicio + duracion);
    osc.connect(ganancia).connect(contexto.destination);
    osc.start(inicio);
    osc.stop(inicio + duracion);
  } catch {
    // Sin audio disponible: la página sigue funcionando en silencio.
  }
}

export const sonido = {
  ronquido() { tono(110, 70, 0.6, "sawtooth", 0, 0.08); tono(90, 60, 0.6, "sawtooth", 0.7, 0.08); },
  cama() { tono(440, 660, 0.15, "triangle"); tono(660, 880, 0.2, "triangle", 0.15); },
  error() { tono(160, 120, 0.3, "square", 0, 0.08); },
  coste() { tono(700, 120, 0.9, "sawtooth", 0, 0.1); },
  barranquilla() { [523, 659, 784, 659].forEach((f, i) => tono(f, f, 0.14, "square", i * 0.14, 0.07)); },
  fanfarria() { [523, 659, 784, 1047].forEach((f, i) => tono(f, f, 0.25, "triangle", i * 0.18)); },
  clic() { tono(900, 700, 0.05, "square", 0, 0.05); },
};

// prepararBotonSonido conecta el botón de silencio de la cabecera.
export function prepararBotonSonido() {
  const boton = document.getElementById("boton-sonido");
  const pintar = () => {
    boton.textContent = silencio ? "🔇 Sin sonido" : "🔊 Con sonido";
    boton.setAttribute("aria-pressed", String(silencio));
  };
  boton.addEventListener("click", () => {
    silencio = !silencio;
    try {
      localStorage.setItem("silencio", silencio ? "1" : "0");
    } catch {
      // Sin almacenamiento: la preferencia dura hasta recargar.
    }
    pintar();
    sonido.clic();
  });
  pintar();
}
