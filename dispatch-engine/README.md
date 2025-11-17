# Dispatch Engine - RideNow POC (Persona 3)

## Descripción

Este microservicio es el **motor de asignación** del sistema RideNow.

Su función principal es:

1.  Escuchar solicitudes de viaje entrantes (desde Google Cloud Pub/Sub).
2.  Buscar el conductor más cercano en tiempo real (usando Redis y GEORADIUS).
3.  **Integrarse funcionalmente con la API de Mapbox para obtener un tiempo estimado de llegada (ETA) y distancia realista.**
4.  Devolver la asignación del conductor con la latencia total medida.

Este componente demuestra el cumplimiento del **Driver Arquitectónico 1: Latencia < 200 ms** y el uso de las tecnologías definidas en el documento final de arquitectura: **Go, Redis, Google Cloud Pub/Sub y Mapbox**.

---

## Tecnologías Utilizadas

*   **Lenguaje**: Go (v1.20+)
*   **Framework Web**: Gin (opcional, usado para posibles endpoints futuros)
*   **Base de Datos en Tiempo Real**: Redis (corriendo localmente en Docker)
*   **Mensajería/Eventos**: Google Cloud Pub/Sub (GCP real)
*   **Ruteo/Mapas**: **Integración funcional con la API de Mapbox**
*   **Plataforma**: Google Cloud Platform (GCP)

---

## Funcionalidad

*   **Conexión a GCP**: Se autentica usando credenciales de una cuenta de servicio y se conecta a Pub/Sub.
*   **Creación de Recursos**: Crea el topic 	rip-requests y la suscripción dispatch-subscription en GCP si no existen.
*   **Procesamiento de Eventos**: Recibe mensajes de solicitud de viaje ({"lat": -34.6037, "lng": -58.3816, "city": "buenos_aires"}).
*   **Consulta a Redis**: Busca conductores cercanos en la ciudad especificada usando GEORADIUS.
*   **Integración Mapbox (Funcional)**: **Realiza una llamada HTTP real a la API de Mapbox** para calcular ETA y distancia realista, usando un token de acceso válido.
*   **Asignación y Medición**: Elige al conductor más cercano y mide la latencia total del proceso.
*   **Publicación de Resultado**: (Opcionalmente) Publica el resultado de la asignación en otro topic o lo devuelve.

---

## Cómo Ejecutar

### Requisitos Previos

*   Go instalado (v1.20 o superior).
*   Docker Desktop instalado y corriendo (para Redis).
*   Cuenta de Google Cloud Platform (GCP) con:
    *   Proyecto creado.
    *   API de Pub/Sub habilitada.
    *   Cuenta de servicio con rol Pub/Sub Editor.
    *   Archivo de credenciales JSON descargado.
*   **Token de acceso de Mapbox** (cuenta gratuita en [https://www.mapbox.com/](https://www.mapbox.com/) - **requerido para que la integración funcione**).

### Pasos

1.  **Configurar Credenciales de GCP**:
    *   Guarda el archivo JSON de credenciales de GCP en una carpeta segura (por ejemplo, C:\Users\TU_NOMBRE\gcp-creds\).
    *   Define la variable de entorno GOOGLE_APPLICATION_CREDENTIALS apuntando al archivo JSON.
        *   En PowerShell: $env:GOOGLE_APPLICATION_CREDENTIALS = "ruta/al/archivo.json"

2.  **Iniciar Redis**:
    *   En una terminal, ejecuta: docker run --name redis-ridenow -p 6379:6379 -d redis:latest

3.  **Cargar Conductores de Prueba (opcional, pero recomendado)**:
    *   Ejecuta el script load_drivers.bat (Windows) o load_drivers.sh (Linux/macOS) ubicado en esta carpeta para precargar ubicaciones de conductores en Redis.

4.  **Configurar Token de Mapbox**:
    *   Abre main.go.
    *   Reemplaza mapboxAccessToken = "pk.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" con **tu token real de Mapbox**.
        *   **¡Importante!** Sin un token válido, la integración con Mapbox **fallará**.

5.  **Ejecutar el Servicio**:
    *   En la carpeta dispatch-engine, ejecuta: go run main.go

El servicio se conectará a Redis y GCP, creará los recursos necesarios y comenzará a escuchar mensajes. **Cuando reciba una solicitud, intentará calcular ETA/distancia usando la API real de Mapbox.**
