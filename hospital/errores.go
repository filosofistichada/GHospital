package hospital

import "errors"

// Errores del hospital. Son variables exportadas para que quien llame pueda
// reconocerlos con errors.Is aunque vengan envueltos con más contexto (%w).
var (
	ErrPacienteNulo        = errors.New("paciente nulo: en este hospital no se atienden fantasmas")
	ErrPacienteDuplicado   = errors.New("el paciente ya estaba admitido: ni que fuera la fila del sancocho para colarse dos veces")
	ErrPacienteNoAdmitido  = errors.New("el paciente no está admitido: que pase primero por la ventanilla")
	ErrPacienteDespierto   = errors.New("el paciente está despierto, bien despierto, mirando el partido del Junior")
	ErrPacienteDormido     = errors.New("el paciente ya está dormido: no se puede roncar más duro")
	ErrPacienteYaEnCama    = errors.New("el paciente ya tiene cama: no sea abusivo, que hay gente en el pasillo")
	ErrPersonalNulo        = errors.New("personal nulo: no se puede contratar a alguien que no existe")
	ErrPersonalDuplicado   = errors.New("esa persona ya trabaja aquí: no le van a pagar dos sueldos")
	ErrSinPersonal         = errors.New("no hay personal disponible: todos están en la tienda viendo al Junior")
	ErrDoctorSinCupo       = errors.New("el doctor no da abasto: ya tiene demasiados pacientes a cargo")
	ErrHabitacionNula      = errors.New("habitación nula: una habitación que no existe es un pasillo con ilusiones")
	ErrHabitacionDuplicada = errors.New("esa habitación ya está registrada")
	ErrCapacidadInvalida   = errors.New("la capacidad debe ser al menos 1: una habitación sin cama es un pasillo con puerta")
	ErrHabitacionLlena     = errors.New("la habitación está llena: ahí ya no cabe ni la moza")
	ErrNoEsOcupante        = errors.New("el paciente no está en esa habitación: el Coste dice que estaba ahí, pero nadie le cree")
	ErrSinHabitacion       = errors.New("no hay cama disponible: el paciente se queda en el pasillo como el novio que llegó tarde a la boda y ya había otro en el altar")
)
