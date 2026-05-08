# AuditForge Proxy

Proxy HTTP/HTTPS nativo expuesto como MCP server.

## Uso

```bash
cd cmd/proxy-server
go run .
```

## Tools MCP

- `proxy.intercept.enable`
- `proxy.intercept.disable`
- `proxy.history.search`
- `proxy.request.get`
- `proxy.request.modify`
- `proxy.request.forward`
- `proxy.request.drop`
- `proxy.stats.get`
- `proxy.findings.list`
- `proxy.export.har`

## Flujo

1. Iniciar el proxy.
2. Configurar browser o app en `localhost:8080`.
3. Habilitar interceptación con filtros.
4. Revisar historial y hallazgos.
5. Exportar HAR si hace falta.

## Notas

- Soporta interceptación HTTP/HTTPS.
- Persiste requests y hallazgos en SQLite.
- No incluye replay diferencial automático.
