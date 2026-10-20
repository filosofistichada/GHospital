package virus

import "errors"

// Errores del juego, comparables con errors.Is.
var (
	ErrNumeroBots          = errors.New("elige de 1 a 3 bots: más gente no cabe en la terraza")
	ErrJuegoTerminado      = errors.New("la partida ya terminó: recojan las sillas y apaguen el picó")
	ErrNoEsBot             = errors.New("este turno no es de un bot: te toca a ti, mija")
	ErrCartaInvalida       = errors.New("esa carta no está en tu mano: ¿la tienes escondida en la manga como el Coste?")
	ErrDescarteInvalido    = errors.New("descarta de 1 a 3 cartas distintas de tu mano")
	ErrTienesCartas        = errors.New("no puedes pasar con cartas en la mano: juega o descarta")
	ErrRivalInvalido       = errors.New("ese rival no existe (y atacarse uno mismo es cosa del Coste)")
	ErrSinPaciente         = errors.New("no hay paciente de ese color ahí")
	ErrColorRepetido       = errors.New("quedarías con dos pacientes del mismo equipo, y eso no se permite")
	ErrColorIncompatible   = errors.New("los colores no combinan: esa carta no le hace efecto a ese hincha")
	ErrPacienteEstable     = errors.New("ese paciente está estable: ni con sancocho se duerme, ni el Coste puede con él")
	ErrCamilleroNoDiagnost = errors.New("El camillero no diagnostica, mijo, llame a un doctor.")
	ErrCosteSinVictimas    = errors.New("El Coste no encontró a quién dormir: ese rival no tiene pacientes que no estén estables")
	ErrNadaQueChismear     = errors.New("no hay chisme: no tienes ataques encima o nadie tiene pacientes libres de ese color")
	ErrNadaQueCambiar      = errors.New("no hay nada que cambiar: los tableros no tienen pacientes que se puedan mover")
)
