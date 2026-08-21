# OSRM Self-Host Setup (Spec 18 Wave A)

Self-host OSRM for production routing. Public demo `http://router.project-osrm.org` is dev-only (1 req/sec, no SLA).

## Quick start (Docker, 2-3 hours first run)

```bash
# 1) Download India OSM extract (~2GB)
wget https://download.geofabrik.de/asia/india-latest.osm.pbf -O india.osm.pbf

# 2) Preprocess (2-4 CPU, 8GB RAM recommended, ~30 min)
docker run -t -v $(pwd):/data osrm/osrm-backend osrm-extract -p /opt/car.lua /data/india.osm.pbf
docker run -t -v $(pwd):/data osrm/osrm-backend osrm-partition /data/india.osrm
docker run -t -v $(pwd):/data osrm/osrm-backend osrm-customize /data/india.osrm

# 3) Serve (port 5000, <100ms per table query)
docker run -d -p 5000:5000 -v $(pwd):/data osrm/osrm-backend osrm-routed --algorithm mld /data/india.osrm

# Verify
curl "http://localhost:5000/table/v1/driving/72.8777,19.0760;72.9781,19.2183?sources=0&destinations=1&annotations=distance,duration"
```

## Env

```
ROUTING_PROVIDER=osrm-selfhost
OSRM_URL=http://osrm.internal:5000
# or ROUTING_PROVIDER=http://osrm.internal:5000
```

Weekly update: re-download `india-latest.osm.pbf` and re-run extract/partition/customize.

## Fallback

Provider `mock` (greedy nearest-neighbor) is used when OSRM is down or `ROUTING_PROVIDER=mock`. `internal/route/optimizer/osrm.go` falls back automatically on network error or non-200.
