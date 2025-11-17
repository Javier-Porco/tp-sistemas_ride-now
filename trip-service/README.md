# Trip Service - RideNow POC (Persona 2)

Este microservicio recibe solicitudes de viaje del frontend, publica un evento en Google Pub/Sub y devuelve una respuesta simulada al frontend.

## Requisitos

- Go 1.19 o superior
- Google Cloud SDK (para el emulador de Pub/Sub)

## Pasos para Ejecutar

1.  **Iniciar el Emulador de Google Pub/Sub (en otra terminal):**
    ```bash
    gcloud beta emulators pubsub start --project=ridenow-poc --host-port=localhost:8085
    ```
    > Asegúrate de que `gcloud` esté instalado y configurado.

2.  **Configurar Variables de Entorno (en la terminal donde correrás el servicio):**
    ```bash
    export PUBSUB_EMULATOR_HOST=localhost:8085
    ```
    En Windows (Git Bash):
    ```bash
    export PUBSUB_EMULATOR_HOST=localhost:8085
    ```

3.  **(Opcional pero recomendado) Crear el Topic en el Emulador:**
    Abre otra terminal, asegúrate de tener la variable `PUBSUB_EMULATOR_HOST` configurada, y ejecuta:
    ```bash
    gcloud pubsub topics create trip-requests --project=ridenow-poc
    ```

4.  **Compilar y Ejecutar el Servicio (en esta carpeta `trip-service`):**
    ```bash
    go run main.go
    ```
    El servicio estará disponible en `http://localhost:8080`.

## Notas

- Este servicio está diseñado para funcionar con el emulador de Pub/Sub local.
- La respuesta actual es simulada para simplificar el POC. En un entorno real, se esperaría una respuesta asincrónica del Dispatch Engine.
- El tiempo de procesamiento incluye el tiempo de publicación en Pub/Sub.
- Se simula un `trace_id` para cumplir con RFN-09.