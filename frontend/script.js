// 🔑 Reemplaza con tu token de Mapbox (debes habilitar Directions API en la consola de Mapbox)
mapboxgl.accessToken = 'pk.eyJ1IjoiamF2aWVyMjAyNSIsImEiOiJjbWh6bjJrbjkwcGZnMmpvcWxmbjRjb3BpIn0.f03glDhGj93LwDg2-JYWUQ';

const map = new mapboxgl.Map({
  container: 'map',
  style: 'mapbox://styles/mapbox/streets-v12',
  center: [-58.3816, -34.6037], // Buenos Aires
  zoom: 12
});

let origin = null;
let destination = null;

const originDisplay = document.getElementById('origin-coords');
const destDisplay = document.getElementById('dest-coords');
const distanceDisplay = document.getElementById('distance');
const etaDisplay = document.getElementById('eta');
const routeInfo = document.getElementById('route-info');
const requestBtn = document.getElementById('requestBtn');
const clearBtn = document.getElementById('clearBtn');
const driverStatus = document.getElementById('driver-status');

// Manejo de clics en el mapa
map.on('click', async (e) => {
  const { lng, lat } = e.lngLat;

  if (!origin) {
    setOrigin({ lng, lat });
  } else if (!destination) {
    setDestination({ lng, lat });
    await fetchRoute();
  } else {
    if (confirm('Ya seleccionaste origen y destino. ¿Quieres cambiar el origen?')) {
      clearSelection();
      setOrigin({ lng, lat });
    }
  }
});

function setOrigin(coord) {
  origin = coord;
  originDisplay.textContent = `${coord.lat.toFixed(4)}, ${coord.lng.toFixed(4)}`;
  new mapboxgl.Popup()
    .setLngLat([coord.lng, coord.lat])
    .setHTML('<b>📍 Origen</b>')
    .addTo(map);
}

function setDestination(coord) {
  destination = coord;
  destDisplay.textContent = `${coord.lat.toFixed(4)}, ${coord.lng.toFixed(4)}`;
  new mapboxgl.Popup()
    .setLngLat([coord.lng, coord.lat])
    .setHTML('<b>📍 Destino</b>')
    .addTo(map);
}

function clearSelection() {
  origin = null;
  destination = null;
  originDisplay.textContent = '—';
  destDisplay.textContent = '—';
  routeInfo.classList.add('hidden');
  requestBtn.disabled = true;
  driverStatus.style.display = 'none';

  // Eliminar la ruta dibujada
  if (map.getLayer('route')) {
    map.removeLayer('route');
    map.removeSource('route');
  }
}

// Botón de limpiar
clearBtn.addEventListener('click', clearSelection);

// Cálculo de ruta con Mapbox Directions API
async function fetchRoute() {
  if (!origin || !destination) return;

  const url = `https://api.mapbox.com/directions/v5/mapbox/driving/${origin.lng},${origin.lat};${destination.lng},${destination.lat}?geometries=geojson&access_token=${mapboxgl.accessToken}`;

  try {
    const response = await fetch(url);
    const data = await response.json();

    if (data.routes && data.routes.length > 0) {
      const route = data.routes[0];
      const distanceKm = (route.distance / 1000).toFixed(1);
      const etaMinutes = Math.ceil(route.duration / 60);

      distanceDisplay.textContent = `${distanceKm} km`;
      etaDisplay.textContent = `${etaMinutes} min`;

      routeInfo.classList.remove('hidden');
      requestBtn.disabled = false;

      // Eliminar capa anterior si existe
      if (map.getLayer('route')) {
        map.removeLayer('route');
        map.removeSource('route');
      }

      map.addLayer({
        id: 'route',
        type: 'line',
        source: {
          type: 'geojson',
          data: route.geometry
        },
        layout: {
          'line-join': 'round',
          'line-cap': 'round'
        },
        paint: {
          'line-color': '#3b82f6',
          'line-width': 6
        }
      });
    }
  } catch (err) {
    console.error('Error al calcular ruta:', err);
    alert('No se pudo calcular la ruta. Intente con otros puntos.');
  }
}

// Botón "Pedir viaje"
requestBtn.addEventListener('click', async () => {
  if (!origin) return;

  requestBtn.disabled = true;
  requestBtn.textContent = 'Buscando conductor...';
  const resultDiv = document.getElementById('result');
  resultDiv.style.display = 'none';
  driverStatus.style.display = 'none';

  try {
    const response = await fetch('http://localhost:8080/trip/request', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: 'user_demo',
        lat: origin.lat,
        lng: origin.lng,
        city: 'buenos_aires'
      })
    });

    if (!response.ok) throw new Error('Error en el backend');

    const data = await response.json();

    // Mostrar resultado
    resultDiv.className = 'success';
    resultDiv.innerHTML = `
      ✅ <strong>¡Asignado!</strong><br>
      Conductor: <code>${data.driver_id}</code><br>
      Distancia al conductor: ${data.distance_km} km<br>
      Latencia: <strong>${data.latency_ms} ms</strong>
    `;
    resultDiv.style.display = 'block';

    // Mostrar trace_id si está presente (cumple RFN-09)
    if (data.trace_id) {
      const traceEl = document.createElement('div');
      traceEl.className = 'trace-id';
      traceEl.innerHTML = `<small>ID de trazabilidad: <code>${data.trace_id}</code></small>`;
      resultDiv.appendChild(traceEl);
    }

    // Mostrar estado del conductor (cumple RFN-10)
    driverStatus.innerHTML = `
      🚕 <strong>Conductor ${data.driver_id} está en camino</strong><br>
      Llegará en ~${Math.max(2, Math.floor(data.distance_km * 2))} min
    `;
    driverStatus.style.display = 'block';
    driverStatus.style.opacity = '0';
    setTimeout(() => {
      driverStatus.style.transition = 'opacity 0.5s';
      driverStatus.style.opacity = '1';
    }, 100);
  } catch (err) {
    resultDiv.className = 'error';
    resultDiv.textContent = 'No se pudo asignar un conductor.';
    resultDiv.style.display = 'block';
    console.error(err);
  } finally {
    requestBtn.textContent = 'Pedir viaje';
  }
});