package clusterlog

import (
	"encoding/json"
	"strings"
)

type inputSchema struct {
	Schema      string                    `json:"$schema"`
	ID          string                    `json:"$id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Type        string                    `json:"type"`
	Required    []string                  `json:"required,omitempty"`
	Properties  map[string]map[string]any `json:"properties"`
	Additional  bool                      `json:"additionalProperties"`
	Examples    []any                     `json:"examples,omitempty"`
}

func property(kind, description string) map[string]any {
	return map[string]any{"type": kind, "description": description}
}

func stringArray(description string) map[string]any {
	return map[string]any{
		"type": "array", "description": description,
		"items": map[string]any{"type": "string"}, "uniqueItems": true,
	}
}

func enumProperty(description string, values ...string) map[string]any {
	items := make([]any, len(values))
	for i, value := range values {
		items[i] = value
	}
	return map[string]any{"type": "string", "description": description, "enum": items}
}

func schemaFor(name string) (inputSchema, bool) {
	base := inputSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		ID:     "https://clusterlog.local/schemas/" + name + ".json",
		Type:   "object", Additional: false,
	}
	switch name {
	case "memory":
		base.Title = "Entrada de memoria operativa"
		base.Description = "Contrato de entrada para clusterlog memory create --from-json. Nunca incluya secretos sin redactar."
		base.Required = []string{"title"}
		base.Properties = map[string]map[string]any{
			"title":            property("string", "Título breve y accionable"),
			"description":      property("string", "Resumen para listados"),
			"tags":             stringArray("Etiquetas temáticas"),
			"systems":          stringArray("Sistemas o componentes afectados"),
			"risk":             enumProperty("Nivel de riesgo", "low", "medium", "high", "critical"),
			"ticket":           property("string", "Ticket, incidente o cambio relacionado"),
			"review_required":  property("boolean", "Incluye la entrada en la cola de revisión"),
			"review_policy":    enumProperty("Quién debe revisar", "all-active", "all-active-except-author"),
			"body_markdown":    property("string", "Cuerpo Markdown completo; sustituye las secciones estructuradas"),
			"context":          property("string", "Contexto de la intervención"),
			"changes":          stringArray("Cambios realizados"),
			"commands":         stringArray("Comandos relevantes con secretos redactados"),
			"validation":       stringArray("Comprobaciones ejecutadas"),
			"rollback":         property("string", "Procedimiento de reversión"),
			"started_at":       property("string", "Inicio RFC3339"),
			"finished_at":      property("string", "Fin RFC3339"),
			"duration_minutes": property("integer", "Duración total en minutos"),
			"agent":            property("string", "Agente que preparó el borrador, por ejemplo claude-code"),
		}
		base.Examples = []any{MemoryInput{
			Title: "Actualizar firmware del switch spine-01", Description: "Cambio controlado y validación de enlaces",
			Tags: []string{"red", "firmware"}, Systems: []string{"spine-01"}, Risk: "high", Ticket: "CHG-1042",
			Changes:    []string{"Respaldé la configuración", "Instalé la versión aprobada"},
			Commands:   []string{"show version", "show interfaces status"},
			Validation: []string{"Todos los uplinks quedaron up", "Sin errores nuevos en logs"},
			Rollback:   "Arrancar la imagen anterior y restaurar la configuración respaldada.", Agent: "claude-code",
		}}
	case "task":
		base.Title = "Tarea compartida"
		base.Description = "Contrato de entrada para clusterlog task create --from-json."
		base.Required = []string{"title"}
		base.Properties = map[string]map[string]any{
			"title":               property("string", "Título de la tarea"),
			"description":         property("string", "Resumen"),
			"project_id":          property("string", "Proyecto relacionado, si existe"),
			"priority":            enumProperty("Prioridad", "low", "medium", "high", "urgent"),
			"assignees":           stringArray("IDs de administradores responsables"),
			"estimate_minutes":    property("integer", "Estimación en minutos"),
			"due":                 property("string", "Fecha límite YYYY-MM-DD"),
			"tags":                stringArray("Etiquetas"),
			"systems":             stringArray("Sistemas afectados"),
			"body_markdown":       property("string", "Cuerpo Markdown completo"),
			"objective":           property("string", "Resultado esperado"),
			"acceptance_criteria": stringArray("Criterios verificables"),
			"notes":               property("string", "Notas de ejecución"),
		}
		base.Examples = []any{TaskInput{Title: "Probar restauración de etcd", Priority: "high", EstimateMinutes: 120, Due: "2026-09-15", Objective: "Validar el RTO del plano de control", Acceptance: []string{"Restauración documentada", "RTO menor a 60 minutos"}}}
	case "project":
		base.Title = "Proyecto de operación"
		base.Description = "Contrato de entrada para clusterlog project create --from-json."
		base.Required = []string{"title", "objective"}
		base.Properties = map[string]map[string]any{
			"title":            property("string", "Nombre del proyecto"),
			"description":      property("string", "Resumen ejecutivo"),
			"objective":        property("string", "Objetivo medible"),
			"owners":           stringArray("IDs de administradores responsables"),
			"target_date":      property("string", "Fecha objetivo YYYY-MM-DD"),
			"status":           enumProperty("Estado", "planned", "active", "blocked", "done", "cancelled", "archived"),
			"tags":             stringArray("Etiquetas"),
			"systems":          stringArray("Sistemas incluidos"),
			"body_markdown":    property("string", "Cuerpo Markdown completo"),
			"scope":            stringArray("Elementos dentro del alcance"),
			"out_of_scope":     stringArray("Elementos fuera del alcance"),
			"milestones":       stringArray("Hitos"),
			"success_criteria": stringArray("Criterios de éxito"),
		}
		base.Examples = []any{ProjectInput{Title: "Estandarizar observabilidad", Objective: "Unificar métricas, logs y alertas del clúster", Status: "planned", Scope: []string{"Inventario de alertas", "Dashboards base"}, Success: []string{"100% de nodos reportando métricas"}}}
	case "content":
		base.Title = "Documento o manual"
		base.Description = "Contrato de entrada para clusterlog content create --from-json."
		base.Required = []string{"section", "title", "body_markdown"}
		base.Properties = map[string]map[string]any{
			"section":       enumProperty("Sección principal", "documentacion", "manuales"),
			"title":         property("string", "Título"),
			"description":   property("string", "Resumen"),
			"tags":          stringArray("Etiquetas"),
			"systems":       stringArray("Sistemas relacionados"),
			"body_markdown": property("string", "Contenido Markdown"),
			"weight":        property("integer", "Orden opcional dentro de la sección"),
		}
		base.Examples = []any{ContentInput{Section: "manuales", Title: "Reemplazo seguro de un nodo de cómputo", Tags: []string{"runbook"}, BodyMarkdown: "## Prerrequisitos\n\n...\n"}}
	default:
		return inputSchema{}, false
	}
	return base, true
}

func (a *App) runSchema(args []string) error {
	if len(args) == 0 {
		return a.printResult("schema", map[string]any{
			"available":     []string{"memory", "task", "project", "content"},
			"usage":         a.Options.Executable + " --json schema memory",
			"stdin_example": "cat input.json | " + a.Options.Executable + " --json --commit memory create --from-json -",
		})
	}
	if len(args) != 1 {
		return NewError(ExitUsage, "schema_usage", "uso: schema [memory|task|project|content]", args)
	}
	name := strings.ToLower(args[0])
	schema, ok := schemaFor(name)
	if !ok {
		return NewError(ExitUsage, "unknown_schema", "esquema desconocido", name)
	}
	// Convertir a un mapa preserva el documento JSON Schema como dato del sobre de salida.
	data, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	return a.printResult("schema."+name, document)
}
