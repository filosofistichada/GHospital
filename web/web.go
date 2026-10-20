// Package web embebe el frontend (HTML, CSS, JS e imágenes) dentro del
// binario, para que "go run . -web" funcione sin archivos sueltos.
package web

import "embed"

// Archivos contiene el frontend; la raíz es esta carpeta.
//
//go:embed index.html juego.html css js img
var Archivos embed.FS
