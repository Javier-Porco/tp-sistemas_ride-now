package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "cloud.google.com/go/pubsub"
    "github.com/gin-gonic/gin"
)

// TripRequest representa la estructura de la solicitud que recibirá el endpoint.
type TripRequest struct {
    UserID string  `json:"user_id"`
    Lat    float64 `json:"lat"`
    Lng    float64 `json:"lng"`
    City   string  `json:"city"`
}

// TripResponse representa la estructura de la respuesta que enviará el endpoint.
type TripResponse struct {
    DriverID   string  `json:"driver_id"`
    DistanceKM float64 `json:"distance_km"`
    LatencyMS  int64   `json:"latency_ms"`
    TraceID    string  `json:"trace_id"` // Añadido para cumplir RFN-09
}

// Variables globales para el cliente de Pub/Sub (simplificado para el POC)
var pubsubClient *pubsub.Client
var tripRequestsTopic *pubsub.Topic

func main() {
    var err error
    ctx := context.Background()
    projectID := "ridenow-poc-jean" // ID de proyecto arbitrario para el emulador

    // Inicializar cliente de Pub/Sub con el emulador
    // NOTA: Asegúrate de que PUBSUB_EMULATOR_HOST esté configurado si usas el emulador
    pubsubClient, err = pubsub.NewClient(ctx, projectID)
    if err != nil {
        log.Fatalf("No se pudo crear el cliente de Pub/Sub: %v", err)
    }
    defer pubsubClient.Close()

    // Obtener el topic trip-requests (debe existir en el emulador)
    topicName := "trip-requests"
    tripRequestsTopic = pubsubClient.Topic(topicName)

    // Configurar Gin
    router := gin.Default()

    // Endpoint POST /trip/request
    router.POST("/trip/request", handleTripRequest)

    // Iniciar el servidor en el puerto 8080
    port := "8080"
    fmt.Printf("Servidor Trip Service escuchando en http://localhost:%s\n", port)
    log.Fatal(router.Run(":" + port))
}

// handleTripRequest maneja la solicitud POST entrante.
func handleTripRequest(c *gin.Context) {
    // Medir el tiempo de inicio para calcular la latencia total del endpoint
    startTime := time.Now()

    var req TripRequest
    // Intentar enlazar el JSON de la solicitud al struct TripRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // Si hay un error en el binding (JSON inválido), devolver error 400
        c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida"})
        return
    }

    // Serializar la solicitud recibida a JSON para publicarla
    eventData, err := json.Marshal(req)
    if err != nil {
        log.Printf("Error al serializar la solicitud para Pub/Sub: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno al procesar la solicitud"})
        return
    }

    // Publicar el mensaje en el topic de Pub/Sub
    result := tripRequestsTopic.Publish(context.Background(), &pubsub.Message{
        Data: eventData,
        // Atributos opcionales del mensaje
        Attributes: map[string]string{
            "source": "trip-service",
            "city":   req.City,
            // Se podría añadir un trace_id aquí también si se genera antes
        },
    })

    // Esperar la confirmación de la publicación (opcional para latencia, pero útil para errores)
    // En un sistema real, esto podría hacerse de forma asíncrona o con reintento.
    _, err = result.Get(context.Background())
    if err != nil {
        log.Printf("Error al publicar en Pub/Sub: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al publicar la solicitud"})
        return
    }

    fmt.Printf("Evento publicado en %s: %s\n", tripRequestsTopic.String(), string(eventData))
	
    // --- Simulación de la respuesta del Dispatch Engine ---
    // En un POC, el Dispatch Engine puede no estar listo aún.
    // Para simular el flujo completo, generamos una respuesta simulada aquí.
    // En un entorno real, este servicio esperaría una respuesta del Dispatch Engine
    // (posiblemente a través de otro topic de respuesta o una llamada HTTP).
    // La latencia medida aquí incluye el tiempo de publicación en Pub/Sub.
    latency := time.Since(startTime).Milliseconds()
    simulatedResponse := TripResponse{
        DriverID:   "driver_simulado_123", // ID simulado
        DistanceKM: 0.85,                 // Distancia simulada
        LatencyMS:  latency,              // Latencia medida
        TraceID:    fmt.Sprintf("sim_trace_%d", time.Now().UnixNano()), // Trace ID simulado para RFN-09
    }
    // --- Fin de la simulación ---

    // Devolver la respuesta simulada (o real, en el futuro) al cliente
    c.JSON(http.StatusOK, simulatedResponse)
}