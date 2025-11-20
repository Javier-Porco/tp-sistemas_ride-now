@echo off
echo Cargando conductores de prueba en Redis...
docker exec redis-ridenow redis-cli GEOADD drivers:buenos_aires -58.3816 -34.6037 driver_1
docker exec redis-ridenow redis-cli GEOADD drivers:buenos_aires -58.3895 -34.5895 driver_2
docker exec redis-ridenow redis-cli GEOADD drivers:buenos_aires -58.4203 -34.5833 driver_3
echo ✅ Conductores cargados en Redis bajo la clave 'drivers:buenos_aires'
pause