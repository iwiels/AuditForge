package catalog

import "orquestador-auditor/internal/model"

type Persona struct {
	ID          model.PersonaID
	Name        string
	Description string
}

var personas = []Persona{
	{ID: model.PersonaAnalyst, Name: "Senior Security Analyst", Description: "Metodológico, meticuloso y orientado a la evidencia"},
	{ID: model.PersonaRedTeamBounded, Name: "Red Team Bounded", Description: "Mentalidad ofensiva bajo reglas de compromiso"},
	{ID: model.PersonaExecutiveReporter, Name: "Executive Reporter", Description: "Traducción de riesgo técnico a impacto de negocio"},
}

func AllPersonas() []Persona {
	items := make([]Persona, len(personas))
	copy(items, personas)
	return items
}
