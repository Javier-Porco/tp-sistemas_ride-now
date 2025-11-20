package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/go-redis/redis/v8"
)

var (
	rdb  *redis.Client
	psub *pubsub.Client
	ctx  = context.Background()
	// Reemplaza ESTE token con tu token de Mapbox real o de prueba
	mapboxAccessToken = "pk.eyJ1IjoiamF2aWVyMjAyNSIsImEiOiJjbWh6bjJrbjkwcGZnMmpvcWxmbjRjb3BpIn0.f03glDhGj93LwDg2-JYWUQ" // OJO: ESTE ES UN EJEMPLO, PON EL TUYO
)

// Estructura para el mensaje recibido desde el Trip Service
type TripRequest struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

// Estructura para la respuesta que enviarás al Trip Service
type AssignmentResponse struct {
	DriverID   string  `json:"driver_id"`
	DistanceKm float64 `json:"distance_km"`
	ETASeconds int     `json:"eta_seconds"` // Simulamos ETA real desde Mapbox
	LatencyMs  int64   `json:"latency_ms"`
}

// Estructura para la respuesta de la API de Mapbox (simplificada)
type MapboxResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"` // en metros
		Duration float64 `json:"duration"` // en segundos
	} `json:"routes"`
}

func main() {
	// 1. Conectar con Redis (que está corriendo en Docker)
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Puerto mapeado de Redis en Docker
	})

	// Verificamos la conexión
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("No se pudo conectar a Redis: %v", err)
	}
	fmt.Println("Conectado a Redis exitosamente!")

	// 2. Conectar con GCP real usando credenciales desde archivo
	// La variable de entorno GOOGLE_APPLICATION_CREDENTIALS debe estar definida
	projectID := "ridenow-poc-jean" // Reemplaza con el ID de tu proyecto real (ej: "ridenow-poc-jean")

	psub, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("No se pudo crear el cliente de Pub/Sub con GCP real: %v", err)
	}
	defer psub.Close()

	fmt.Println("Conectado al servicio real de Pub/Sub en GCP exitosamente!")

	// 3. Definir el nombre del topic y la suscripción
	topicName := "trip-requests"
	subscriptionName := "dispatch-subscription"

	// Creamos el topic si no existe (en GCP REAL)
	topicObj := psub.Topic(topicName)
	exists, err := topicObj.Exists(ctx)
	if err != nil {
		log.Printf("Error al verificar si el topic existe en GCP: %v", err)
	}
	if !exists {
		fmt.Printf("El topic %s no existe en GCP. Creándolo...\n", topicName)
		topicObj, err = psub.CreateTopic(ctx, topicName)
		if err != nil {
			log.Fatalf("No se pudo crear el topic en GCP: %v", err)
		}
		fmt.Printf("Topic %s creado exitosamente en GCP.\n", topicName)
	} else {
		fmt.Printf("Topic %s ya existe en GCP.\n", topicName)
	}

	// Creamos la suscripción si no existe (en GCP REAL)
	sub := psub.Subscription(subscriptionName)
	exists, err = sub.Exists(ctx)
	if err != nil {
		log.Fatalf("Error al verificar si la suscripción existe en GCP: %v", err)
	}
	if !exists {
		fmt.Printf("Creando suscripción %s al topic %s en GCP...\n", subscriptionName, topicName)
		config := pubsub.SubscriptionConfig{
			Topic: topicObj,
		}
		sub, err = psub.CreateSubscription(ctx, subscriptionName, config)
		if err != nil {
			log.Fatalf("No se pudo crear la suscripción en GCP: %v", err)
		}
		fmt.Printf("Suscripción %s creada exitosamente en GCP.\n", subscriptionName)
	} else {
		fmt.Printf("Suscripción %s ya existe en GCP.\n", subscriptionName)
	}

	// 4. Definir la función que manejará los mensajes recibidos
	messageHandler := func(ctx context.Context, msg *pubsub.Message) {
		fmt.Printf("Mensaje recibido (raw): %s\n", string(msg.Data))

		// --- Parsear el mensaje JSON recibido ---
		var request TripRequest
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			log.Printf("Error al parsear el mensaje JSON: %v", err)
			msg.Nack() // Indicamos que no se pudo procesar, para reintento
			return
		}

		fmt.Printf("Solicitud recibida para (%f, %f) en %s\n", request.Lat, request.Lng, request.City)

		// Medimos el tiempo total del proceso de asignación (desde que recibe el evento)
		startTotal := time.Now()

		// 5. Buscar conductores cercanos usando GEORADIUS
		//    La clave en Redis es 'drivers:CIUDAD'
		key := fmt.Sprintf("drivers:%s", request.City)

		// Definimos un radio de búsqueda (en metros)
		radius := 2000.0 // 2 km

		// Medimos solo el tiempo de la operación de Redis
		startRedis := time.Now()

		// GEORADIUS: Busca puntos geográficos en un radio
		locations, err := rdb.GeoRadius(ctx, key, request.Lng, request.Lat, &redis.GeoRadiusQuery{
			Radius:    radius,
			Unit:      "m", // metros
			WithCoord: true,
			WithDist:  true,
			Sort:      "ASC", // Ordena por distancia ascendente (el más cercano primero)
		}).Result()

		redisLatency := time.Since(startRedis)

		if err != nil {
			log.Printf("Error al buscar conductores en Redis: %v", err)
			msg.Nack() // Indicamos que no se pudo procesar, para reintento
			return
		}

		if len(locations) == 0 {
			fmt.Println("No se encontraron conductores cercanos para la solicitud.")
			// Enviar evento de "no disponible" o manejar según el sistema real
			// Por ahora, solo terminamos.
			totalLatency := time.Since(startTotal)
			fmt.Printf("--- Procesamiento completado (sin conductor) en %v (Redis: %v) ---\n", totalLatency, redisLatency)
			msg.Ack() // Avisamos que el mensaje fue procesado
			return
		}

		// 6. Elegimos el conductor más cercano (está primero por el Sort: ASC)
		closestDriver := locations[0]

		// --- Simular integración con Mapbox real ---
		// Supongamos que las coordenadas del conductor están en `closestDriver.Coordinate`
		driverLat := closestDriver.Latitude  // <-- CORRECTO PARA TU VERSION DE go-redis
		driverLng := closestDriver.Longitude // <-- CORRECTO PARA TU VERSION DE go-redis
		// Medimos el tiempo de la simulación de la API de Mapbox
		startMapbox := time.Now()
		etaSeconds, distanceKm, err := getETAFromMapbox(request.Lat, request.Lng, driverLat, driverLng)
		mapboxLatency := time.Since(startMapbox)

		if err != nil {
			log.Printf("Error al obtener ETA/distance desde Mapbox: %v", err)
			// Si Mapbox falla, podemos usar la distancia de Redis como fallback
			etaSeconds = int(closestDriver.Dist * 0.1) // Aproximación burda
			distanceKm = closestDriver.Dist / 1000.0
		}

		// Calculamos la latencia total del proceso
		totalLatency := time.Since(startTotal)

		// Construimos la respuesta con datos reales de Mapbox (o fallback)
		response := AssignmentResponse{
			DriverID:   closestDriver.Name,
			DistanceKm: distanceKm,
			ETASeconds: etaSeconds,
			LatencyMs:  totalLatency.Milliseconds(),
		}

		// Convertimos la respuesta a JSON
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error al convertir la respuesta a JSON: %v", err)
			msg.Nack()
			return
		}

		fmt.Printf("Conductor asignado: %s\n", response.DriverID)
		fmt.Printf("Distancia: %.2f km (de Mapbox o Redis)\n", response.DistanceKm)
		fmt.Printf("ETA: %d segundos (de Mapbox)\n", response.ETASeconds)
		fmt.Printf("Latencia total: %d ms\n", response.LatencyMs)
		fmt.Printf("Latencia Redis: %v\n", redisLatency)
		fmt.Printf("Latencia Mapbox (simulada): %v\n", mapboxLatency)
		fmt.Printf("Respuesta JSON: %s\n", string(jsonResponse))

		// --- CÓDIGO DE DEVOLUCIÓN DE RESULTADO ---
		// AQUÍ es donde vos y tus compañeros deciden cómo se envía la respuesta.
		// OPCIÓN 1: Enviar por HTTP al Trip Service (como en el código anterior)
		//           (Tu compañero debe tener un endpoint POST /trip/response)
		// import "bytes" y "net/http" arriba si usas esta opción
		/*
		   url := "http://localhost:8080/trip/response"
		   resp, err := http.Post(url, "application/json", bytes.NewReader(jsonResponse))
		   if err != nil {
		       log.Printf("Error al enviar la respuesta al Trip Service: %v", err)
		       msg.Nack()
		       return
		   }
		   defer resp.Body.Close()

		   if resp.StatusCode != http.StatusOK {
		       log.Printf("El Trip Service respondió con status %d", resp.StatusCode)
		       msg.Nack()
		       return
		   }

		   fmt.Printf("--- Asignación completada y respuesta enviada al Trip Service ---\n")
		*/

		// OPCIÓN 2: Enviar por otro topic de Pub/Sub (asíncrono)
		//           (Tu compañero debe estar suscrito a este topic)
		// Comentamos esta parte por ahora, pero es una alternativa.
		/*
		   resultTopicName := "trip-results" // Define el nombre con tu compañero
		   resultTopic := psub.Topic(resultTopicName)
		   resultExists, err := resultTopic.Exists(ctx)
		   if err != nil {
		       log.Printf("Error al verificar si el topic de resultados existe en GCP: %v", err)
		       msg.Nack()
		       return
		   }
		   if !resultExists {
		       log.Printf("El topic de resultados %s no existe en GCP.", resultTopicName)
		       msg.Nack()
		       return
		   }

		   resultMsg := &pubsub.Message{
		       Data: jsonResponse,
		   }

		   resultTopic.Publish(ctx, resultMsg).Get(ctx)
		   fmt.Printf("--- Asignación completada y respuesta publicada en %s ---\n", resultTopicName)
		*/

		// Por ahora, como no sabemos qué opción van a elegir, solo imprimimos
		// que el proceso terminó y que el mensaje fue procesado.
		fmt.Printf("--- Procesamiento de asignación finalizado ---\n")

		// Avisamos que el mensaje fue procesado exitosamente
		msg.Ack()
	}
	// 5. Iniciar la escucha de mensajes en la suscripción
	fmt.Println("Esperando mensajes en la suscripción de GCP...")
	fmt.Println("Para salir, presiona Ctrl+C")
	err = sub.Receive(ctx, messageHandler)
	if err != nil {
		log.Fatalf("Error recibiendo mensajes desde GCP: %v", err)
	}

	// Esperar señal de interrupción (Ctrl+C) para salir limpiamente
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nSaliendo...")
}

// Simula una llamada a la API de Mapbox para obtener distancia y ETA
func getETAFromMapbox(pickupLat, pickupLng, driverLat, driverLng float64) (int, float64, error) {
	// Este es un ejemplo de cómo sería una llamada real a Mapbox
	// Reemplaza 'mapboxAccessToken' con tu token real
	// Endpoint de ejemplo: directions
	url := fmt.Sprintf("https://api.mapbox.com/directions/v5/mapbox/driving/%f,%f;%f,%f?access_token=%s&geometries=geojson&steps=false", pickupLng, pickupLat, driverLng, driverLat, mapboxAccessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("error al llamar a la API de Mapbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("la API de Mapbox respondió con status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("error al leer la respuesta de Mapbox: %w", err)
	}

	var mapboxResp MapboxResponse
	if err := json.Unmarshal(body, &mapboxResp); err != nil {
		return 0, 0, fmt.Errorf("error al parsear la respuesta de Mapbox: %w", err)
	}

	if len(mapboxResp.Routes) == 0 {
		return 0, 0, fmt.Errorf("no se encontraron rutas en la respuesta de Mapbox")
	}

	// Devolvemos la duración (ETA en segundos) y la distancia (en km)
	etaSeconds := int(mapboxResp.Routes[0].Duration)
	distanceKm := mapboxResp.Routes[0].Distance / 1000.0

	return etaSeconds, distanceKm, nil
}
