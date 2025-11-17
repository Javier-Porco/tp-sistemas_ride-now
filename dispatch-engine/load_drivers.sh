#!/bin/bash

# Este script carga conductores ficticios en Redis
echo "Cargando conductores de prueba en Redis..."

# Conductor 1 - Cerca del Obelisco, Buenos Aires
redis-cli GEOADD drivers:buenos_aires -58.3816 -34.6037 driver_1

# Conductor 2 - Recoleta
redis-cli GEOADD drivers:buenos_aires -58.3895 -34.5895 driver_2

# Conductor 3 - Palermo
redis-cli GEOADD drivers:buenos_aires -58.4203 -34.5833 driver_3

echo "✅ Conductores cargados en Redis bajo la clave 'drivers:buenos_aires'"