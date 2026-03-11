# KubeDoctor - Quick Start Guide

## 🚀 Inicio Rápido (2 minutos)

### 1. Compilar

```bash
go build -o kubedoctor
```

### 2. Ejecutar

```bash
./kubedoctor
```

Eso es todo! KubeDoctor se conectará automáticamente a tu contexto actual de kubectl.

---

## 📋 Guía de Navegación

### Vista de Deployments

```
🚀 KubeDoctor - mas-billing-prod @ mas-billing-prod-es

📦 DEPLOYMENTS (8)

▶ api-gateway                              (3/3) ✅
  billing-service                          (2/3) ⚠️
  payment-processor                        (0/3) ❌
```

**Teclas:**
- `j` o `↓` - Bajar
- `k` o `↑` - Subir
- `Enter` - Ver pods del deployment
- `n` - Cambiar namespace ⭐ NUEVO
- `r` - Refrescar lista
- `q` - Salir

### Vista de Pods

```
📦 billing-service - Pods

PODS:

▶ billing-service-7d9f8-xkj2p   CrashLoopBackOff   [15 restarts] 🔄💥
  billing-service-7d9f8-abc123  Running            [0 restarts]  ✅
```

**Teclas:**
- `j` o `↓` - Bajar
- `k` o `↑` - Subir  
- `d` o `Enter` - Diagnosticar pod
- `Esc` - Volver a deployments

### Vista de Diagnóstico (KubeDoctor)

```
🔍 Diagnosis: billing-service-7d9f8-xkj2p

Health Score: 25/100

Root Cause: PostgreSQL Connection Refused

🔴 CRITICAL ISSUES:

  1. PostgreSQL Connection Refused: connection refused postgres:5432
     💡 Cannot connect to PostgreSQL database...
     
📋 RECOMMENDED ACTIONS:

  1. 🔴 Verify database service is running
  2. 🔴 Check DATABASE_URL environment variable
  3. 🔴 Verify service endpoints

🛠️  DEBUG COMMANDS:

  1. kubectl get svc -n mas-billing-prod | grep postgres
  2. kubectl get endpoints -n mas-billing-prod | grep postgres
```

**Teclas:**
- `Esc` - Volver a pods
- `q` - Salir

---

## 🔧 Configuración

### Cambiar Namespace

**⭐ NUEVA FORMA - Desde dentro de KubeDoctor:**

1. Ejecuta `./kubedoctor`
2. En la vista de deployments, presiona `n`
3. Navega por la lista de namespaces (j/k)
4. Presiona `Enter` para cambiar
5. ¡Listo! Los deployments se cargan automáticamente

**Forma tradicional - Con kubectl:**

```bash
# Ver namespace actual
kubectl config view --minify | grep namespace

# Cambiar namespace
kubectl config set-context --current --namespace=otro-namespace

# Ejecutar KubeDoctor
./kubedoctor
```

### Cambiar Contexto

```bash
# Listar contextos
kubectl config get-contexts

# Cambiar contexto
kubectl config use-context otro-contexto

# Ejecutar KubeDoctor
./kubedoctor
```

---

## 🎯 Casos de Uso durante Guardia

### Caso 1: "El deployment está caído"

1. Ejecuta `./kubedoctor`
2. Busca el deployment con indicador ❌ (rojo)
3. Presiona `Enter` para ver pods
4. Selecciona pod con error y presiona `d`
5. Lee el "Root Cause" y "Recommended Actions"
6. Copia los "Debug Commands" y ejecútalos

### Caso 2: "Algunos pods no arrancan"

1. Ejecuta `./kubedoctor`
2. Busca el deployment con indicador ⚠️ (amarillo)
3. Presiona `Enter` para ver pods
4. Identifica pods con status `Pending`, `CrashLoopBackOff`, etc.
5. Selecciona y presiona `d` para diagnosis
6. Sigue las sugerencias de KubeDoctor

### Caso 3: "El pod se reinicia constantemente"

1. Ejecuta `./kubedoctor`
2. Entra al deployment
3. Busca pods con alto número de restarts (ej: [15 restarts])
4. Presiona `d` para diagnosis
5. KubeDoctor analizará:
   - Exit codes (137 = OOMKilled)
   - Logs para patrones de error
   - Probes de liveness/readiness
6. Te dará sugerencias específicas

---

## 🩺 Patrones de Error Detectados

KubeDoctor detecta automáticamente:

### Memoria
- ✅ OOMKilled (Exit 137)
- ✅ Memory allocation errors
- ✅ Memory exhaustion

### Red/Conectividad
- ✅ Database connection failures (PostgreSQL, MySQL, MongoDB)
- ✅ Connection timeouts
- ✅ DNS failures

### Configuración
- ✅ ConfigMaps missing
- ✅ Secrets missing
- ✅ Environment variables missing
- ✅ Files not found

### Imágenes
- ✅ ImagePullBackOff
- ✅ Image not found
- ✅ Registry authentication

### Health Checks
- ✅ Liveness probe failures
- ✅ Readiness probe failures

### Recursos
- ✅ Insufficient CPU/memory
- ✅ Pod evictions
- ✅ CPU throttling

---

## 💡 Tips

### Ver solo errores críticos

Cuando entres a diagnóstico, enfócate en la sección **🔴 CRITICAL ISSUES** primero.

### Copiar comandos kubectl

Los comandos en la sección **🛠️ DEBUG COMMANDS** están listos para copiar y pegar. Ya tienen reemplazados:
- `<pod-name>` → nombre del pod
- `<namespace>` → tu namespace
- `<deployment>` → nombre del deployment

### Refresh rápido

Presiona `r` en la vista de deployments para refrescar sin salir de la app.

### Navegación rápida

Usa `j` y `k` (estilo vim) - es más rápido que las flechas si tienes muchos deployments/pods.

---

## 🐛 Troubleshooting

### "Error initializing KubeDoctor"

```bash
# Verifica conexión al cluster
kubectl cluster-info

# Verifica que puedes listar deployments
kubectl get deployments

# Verifica tu kubeconfig
ls -la ~/.kube/config
```

### "No deployments found"

```bash
# Verifica namespace
kubectl config view --minify | grep namespace

# Lista deployments en el namespace
kubectl get deployments -n <tu-namespace>

# Cambia namespace si es necesario
kubectl config set-context --current --namespace=<namespace>
```

### La aplicación no responde

- Presiona `Ctrl+C` para forzar salida
- Puede estar cargando datos (cluster lento)
- Timeouts configurados a 15 segundos

---

## 📚 Más Información

Ver [README_KUBEDOCTOR.md](./README_KUBEDOCTOR.md) para documentación completa.

---

**¡Feliz debugging! 🩺**
