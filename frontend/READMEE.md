# RideNow – Caso Testigo (Frontend)

Este repositorio contiene el **frontend interactivo** del **caso testigo** del sistema **RideNow**, desarrollado como parte del Trabajo Práctico de *Ingeniería del Software III*.

El objetivo es demostrar que la arquitectura propuesta —con **Go**, **Redis**, **Google Pub/Sub**, **Mapbox** y **GCP**— permite cumplir con los **requisitos funcionales y no funcionales críticos**, especialmente:

- **RF-04**: Integración con servicio de mapas para rutas y ETA  
- **RFN-09**: Trazabilidad mediante `trace_id`  
- **RFN-10**: Actualización de estado en <1 segundo  
- **Driver 1**: Latencia de asignación <200 ms  

> ✅ **Este frontend se integra con los microservicios `trip-service` y `dispatch-engine` desarrollados en Go.**

---

## 🌟 Características

- 🗺️ **Mapa interactivo con Mapbox** (tecnología elegida en el documento, sección 3.7)  
- 📍 Selección de **origen y destino** mediante clics en el mapa  
- 📏 Cálculo en tiempo real de **distancia y ETA** usando **Mapbox Directions API**  
- 🚕 Simulación de **asignación de conductor** con animación visual  
- 🔍 Muestra el **`trace_id` único** de cada solicitud (**RFN-09**)  
- 🧹 Botón para **limpiar selección** y reiniciar el flujo  
- 🖥️ Diseño responsivo y limpio, listo para demostración

---

## 🛠️ Requisitos

- Tener el **backend (`trip-service`)** corriendo en `http://localhost:8080`
- Contar con un **token de Mapbox** con **Directions API habilitada**
  - [Crear una cuenta gratuita en Mapbox](https://account.mapbox.com/)
  - Habilitar **"Mapbox Directions API"** en *APIs > Directions*

---

## ⚙️ Configuración

1. Cloná el repositorio:
   ```bash
   git clone https://github.com/Javier-Porco/tp-sistemas_ride-now.git
   cd tp-sistemas_ride-now
