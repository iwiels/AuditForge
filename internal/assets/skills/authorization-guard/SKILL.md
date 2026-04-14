# Skill: Authorization Guard

**Categoría:** safety  
**Cuándo activar:** automáticamente antes de cualquier acción activa sobre un target

---

## Protocolo

### Paso 1 — Verificación de autorización via memoria

Antes de iniciar cualquier análisis activo, ejecutá:

```
memory.search("authorized engagement scope")
memory.search("audit session target")
```

**Si encontrás una entrada reciente (misma sesión o mismo día) que incluye:**
- El target actual
- Una indicación de autorización ("authorized", "in scope", "engagement")
- Fecha coherente

→ Logeá "✓ Autorización verificada via memoria" y procedé.

**Si NO encontrás contexto:**
→ Solicitá autorización explícita antes de continuar. Ver Paso 3.

### Paso 2 — Verificación de alcance

Con la autorización confirmada, verificá los límites:

```
¿El target actual es el autorizado?
  Autorizado: app.empresa.com
  ¿Puedo tocar: api.empresa.com? → NO sin confirmación
  ¿Puedo tocar: empresa.com?    → NO sin confirmación
  ¿Puedo tocar: staging.app.empresa.com? → DEPENDE del scope

¿La acción que voy a realizar está dentro del scope?
  Autorizado: "auditoría web de la aplicación"
  ¿Puedo: revisar el código fuente si está en scope? → SÍ
  ¿Puedo: atacar la infraestructura de base? → NO
  ¿Puedo: escalar a sistemas internos? → NO sin confirmación
```

**Si la acción propuesta está fuera del scope:**
→ Detenerse. Documentar el hallazgo como "fuera de alcance — requiere confirmación". Notificar al usuario.

### Paso 3 — Solicitud de autorización explícita

Cuando no hay contexto de autorización en memoria:

```
⚠️ AUTORIZACIÓN REQUERIDA

Antes de iniciar el análisis de [TARGET], necesito confirmar:

1. ¿Tenés autorización explícita para realizar esta auditoría?
   (contratos, letter of engagement, acuerdo escrito con el propietario)

2. ¿Cuál es el alcance exacto?
   □ Solo la aplicación web principal
   □ Todos los subdominios
   □ Infraestructura y APIs
   □ Código fuente (si está disponible)
   □ Pipeline CI/CD

3. ¿Hay restricciones específicas?
   (no tocar el entorno de producción, no autenticarse con cuentas reales, etc.)

Una vez confirmado, guardá esta autorización en memoria para que el equipo la herede.
```

### Paso 4 — Registro de autorización en memoria

Al recibir autorización, guardala en memoria con:

```
memory.save({
  kind: "authorization",
  title: "Engagement authorization — [TARGET]",
  body: "Target: [target]. Scope: [scope]. Restricciones: [restricciones]. Autorizado por: [quien]. Fecha: [fecha].",
  tags: ["authorized", "engagement", "scope", target]
})
```

### Paso 5 — Límites absolutos (no negociables)

Sin importar la autorización recibida, estos límites son absolutos:

```
❌ Modificar datos en producción
❌ Eliminar archivos o registros
❌ Ejecutar exploits que causen downtime
❌ Perscar cambios en el sistema target
❌ Acceder a datos personales de usuarios reales más allá de lo mínimo para confirmar el vector
❌ Moverse lateralmente a sistemas fuera del scope sin confirmación explícita
❌ Usar credenciales reales de usuarios que no sean cuentas de test
```

### Herencia de autorización para sub-agentes

Cuando el orchestrator delega a un sub-agente, el sub-agente hereda la autorización. El sub-agente NO debe:
- Pedir autorización nuevamente al usuario
- Rechazar tareas porque "no tiene autorización" si fue invocado por el orchestrator

El sub-agente SÍ debe:
- Verificar que la tarea está dentro del scope autorizado
- Detenerse y escalar si se le pide actuar fuera del scope

---

## Anti-patterns

- ❌ Iniciar análisis activo sin verificar autorización primero
- ❌ Asumir que una solicitud del usuario implica autorización del target
- ❌ No registrar la autorización en memoria — los sub-agentes no tendrán contexto
- ❌ Tratar los límites absolutos como negociables con argumentos de "es solo una prueba"
